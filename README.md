# 分布式对象存储系统
# Distributed-object-storage-system

## 1、接口和数据存储分离的架构

![1](./image/1.png)

该架构分为三层：

**接口服务层**：提供了对外的REST接口

**中间件**：

* **RabbitMQ**：负责心跳包和消息的传输

&emsp;&emsp;​	--`ApiServer exchange`：用于心跳包的传输。所有**接口服务节点**绑定该`exchange`，所有发往该`exchange`的消息都会被转发给绑定它的所有消息队列

&emsp;&emsp;​	--`DataServer exchange`：用于定位消息的传输。所有**数据服务节点**绑定该exchange，用于接收接口服务的定位消息。
* **Elastic Search**:负责存储元数据

**数据服务层**：提供了数据的存储功能

## 2、基础环境搭建以及配置
绑定一台服务器上的多个ip地址，配置六个数据服务节点和两个接口服务节点：
```
ifconfig enp3s0:1 10.29.1.1/16
ifconfig enp3s0:2 10.29.1.2/16
ifconfig enp3s0:3 10.29.1.3/16
ifconfig enp3s0:4 10.29.1.4/16
ifconfig enp3s0:5 10.29.1.5/16
ifconfig enp3s0:6 10.29.1.6/16
ifconfig enp3s0:7 10.29.2.1/16
ifconfig enp3s0:8 10.29.2.1/16
```
RabbitMQ采用docker容器的方式部署，根据本地容器的相关信息，其ip和端口信息如下：
```
172.17.0.2：5672
```
ElasticSearch也以docker容器方式部署，但由于本机硬件限制，所以部署在内网另一台服务器上，其ip和端口信息如下：
```
192.168.178.176：9200
```
此处的配置信息为
```
export ES_SERVER=192.168.178.176:9200
export RABBITMQ_SERVER=amqp://test:test@172.17.0.2:5672
```
数据服务启动时（以数据服务节点3为例子），`STORAGE_ROOT`为存储根目录
```
LISTEN_ADDRESS=10.29.1.3:12345 STORAGE_ROOT=/tmp/3 go run dataServer/dataServer.go
```
接口服务启动时（以接口服务节点1为例子）
```
LISTEN_ADDRESS=10.29.2.1:12345 go run ApiServer/ApiServer.go
```

## 3、对于心跳信息的处理

#### 从接口服务（`ApiServer/heartbeat`）的角度看：

设置了一个`map[string]time.Time`类型的`dataServers`变量存储存活的数据服务节点。接收心跳信息刷新每个已注册数据服务节点信息的存活时间，移除超时的数据服务节点相关信息。

#### 从数据服务（`dataServer/heartbeat`）的角度看：

每隔5秒向`RabbitMQ`的`apiServers exchange`发送一次心跳信息，即向所有接口服务节点注册自己的存在。

![1](./image/2.png)

## 4、数据服务节点启动
除了心跳信息的发送，数据服务在启动时还执行两个操作
### 4.1、存储对象收集
&emsp;&emsp;数据服务节点首先在本地磁盘上自检，将本地磁盘上存储的对象名称(hash值)收集到本地的内存中，
在此我们创建了一个`map[string]int`的映射结构，并将其作为数据服务节点的对象池。

![c](./image/4-1.png)

&emsp;&emsp;采用这种方式具有优缺点：
* 优点是：在未来我们需要查询某一对象具体存放位置时，无需迫使数据服务节点频繁的访问其磁盘，大大加快了未来查询操作的时间。
* 缺点是：导致数据服务节点开机启动时间过长，启动后过一段时间才能正常工作。

### 4.2、定位服务启动
&emsp;&emsp;首先创建一个`rabbitmq.RabbitMQ`结构体,并将其绑定到`dataServer Exchange`上。
```
q := rabbitmq.New(os.Getenv("RABBITMQ_SERVER"))
q.Bind("dataServers")
```
&emsp;&emsp;这样，数据服务节点就能接收到所有从接口服务节点传送来的对象`hash值`，
并通过查询本地对象池中是否存在该对象`hash值`，
从而返回自身的`ip地址`。

### 对于PUT操作：

1、由于加入了数据校验过程，所以在传输数据之前我们计算数据的hash值，假设我们现在传输的对象名为`test4`,
对象内容为`this object will have only 1 instance`,
通过对象内容计算hash值为`aWKQ2BipX94sb+h3xdTbWYAu1yzjn5vyFG2SOwUQIXY=`
```bash
echo -n "this object will have only 1 instance" | openssl dgst -sha256 -binary | base64
```
所以，我们PUT操作的命令为
```
curl -v 10.29.2.1:12345/objects/test4 -XPUT -d"this object will have only 1 instance" -H "Digest: SHA-256=aWKQ2BipX94sb+h3xdTbWYAu1yzjn5vyFG2SOwUQIXY="
```

### 对于GET操作：

1、命令`curl 10.29.2.1:12345/objects/test2`,`10.29.2.1`为接口服务节点。

2、首先进行定位，获取对象名`test2`，并向`DataServer exchange`群发`test2`名字。

3、数据服务节点自查本地内容，如果本地存在，通过定位程序（`dataServer/locate/dataserver.go`）中第25行的`Send`函数向消息的发送方返回本服务节点的监听地址。假设为`10.29.1.1：12345`

4、替换HTTP包中的ip地址，即`10.29.2.2:12345 -> 10.29.1.1：12345`。此时就会通过HTTP协议获取数据服务节点上的地址，并通过io流显示给用户。

### 元数据服务

新增功能接口：
* GET /objects/<object_name>?version=<version_id> : 获取指定版本的对象
* PUT /objects/<object_name> : 推送对象
  * 将对象散列值和长度作为元数据保存在元数据服务中
  * PUT成功后，会为该对象添加一个新版本，版本号从1开始。、
* DELETE /objects/<object_name> : 在删除一个对象时，只需要给对象添加一个表示删除的特殊版本，数据是仍然保留在数据节点上的。
* GET /versions/ : 查询所有对象的版本
* GET /versions/<object_name> : 查询指定对象的版本

### 数据校验和去重
新增REST接口：
* POST /temp/<hash>:POST方法访问temp接口，该方式会在数据服务节点上创建一个临时对象，并返回一个uuid来标识该临时对象
* PATCH /temp/<uuid>：用于访问数据服务节点上的临时对象
* PUT /temp/<uuid>：接口服务数据校验一致，调用PUT方法将临时文件转正
* DELETE /temp/<uuid>：接口服务数据校验不一致，调用DELETE方法删除该临时文件



### 优势：

1、实现了接口服务与数据服务分离的架构，任意新主机只需要向`RabbitMQ`注册，即可获取数据服务的支持。或是作为数据服务节点，承载数据服务的能力。

### 不足：

1、由于`PUT`时随机选择数据服务节点，这就导致多次`PUT`相同内容会导致不同的数据节点上存储相同的内容。这里需要**数据去重**。（如果做了去重，还需要进行容灾）

2、版本控制问题。现在的分布式存储系统无法做到对于同一份数据不同版本的切换，这里需要用到**元数据服务**。

