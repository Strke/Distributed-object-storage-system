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

&emsp;&emsp;设置了一个`map[string]time.Time`类型的`dataServers`变量存储存活的数据服务节点。
接收心跳信息刷新每个已注册数据服务节点信息的存活时间，移除超时的数据服务节点相关信息。

#### 从数据服务（`dataServer/heartbeat`）的角度看：

&emsp;&emsp;每隔5秒向`RabbitMQ`的`apiServers exchange`发送一次心跳信息，即向所有接口服务节点注册自己的存在。

![1](./image/2.png)

## 4、数据服务节点启动
&emsp;&emsp;除了心跳信息的发送，数据服务在启动时还执行两个操作
### 4.1、存储对象收集

&emsp;&emsp;由于该系统上实现了数据冗余，所以每一个数据服务节点上存储了其中一个分片，
我们在每一个数据服务节点上都有一个对象池来存储对象名称(hash值)及其分片id的映射关系。

&emsp;&emsp;因为我们会将hash所代表的完整对象分割为4+2份，因此该分片id表示当前数据服务节点上的文件是其中的第几份

&emsp;&emsp;数据服务节点首先在本地磁盘上自检，将本地磁盘上对象hash值及其分片id的映射关系收集到本地的内存map中。

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
从而返回自身的`ip地址`以及本地的`分片id`。

## 5、PUT操作：

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
2、从HTTP请求中提取出对象的`size`、`hash`信息

3、根据对象的`hash`信息，我们对其进行**去重**判断。将`hash`信息经由`dataServer Exchange`转发到各个数据服务节点上，
由各个数据服务节点查询本地对象池。若查询到则返回`200`状态码，向elasticsearch数据库中添加新版本元数据并结束后续操作。

4、传输对象的分片信息

&emsp;&emsp;若对象不存在于数据服务节点上，则接口服务节点从自身心跳信息服务的对象池中随机获取 `4+2`个数据服务节点`ip`，
这里的4+2指的是为了实现数据冗余，4个数据服务节点用来存储数据分片，2个数据服务节点用来存储校验分片

&emsp;&emsp;每个分片的传输流程都相当类似，我们选择分片0作为例子讲解：
我们遍历选取的六个数据服务节点`ip地址`，按照遍历顺序，我们向第一个ip发送分片0的POST请求，请求创建一个`hash.分片id`的对象。
这个过程会在该数据服务节点上随机创建一个uuid，
并且将该对象的uuid，name,size作为内容保存在临时存储区的一个名为`uuid`的临时文件中。
然后再创建一个`uuid.dat`文件用于后面接收文件内容。

5、传输对象分片内容

&emsp;&emsp;接下来我们通过`PATCH`方法传输对象的内容，传输完成后将对象内容大小与对象信息文件中的`Size`进行对比，从而判断文件是否出错。

![c](./image/5.png)


5、传输文件的同时，接口服务节点对该文件的hash值进行计算,得到结果d，并与用户传进来的hash值进行比较。
* 若`d == hash`,则将我们刚刚传输到数据服务节点临时存储区域上的对象转移到正式存储区域`objects/`，
并将文件名更改为其hash值，且删除掉临时存储区域`temp/`上的uuid.dat和uuid两个临时文件。
并且同步数据服务节点上的对象池

* 若不相等，则删除该uuid.dat和uuid两个临时文件

6、在成功将对象存储到数据服务节点的正式存储区域后，我们还需要同步ElasticSearch上的元数据。
在这里，我们将(name, hash, size）存放到ElasticSearch中，通过SearchLatestVersion()函数，我们查询到名字为`name`
的最新元数据，将其版本号`Version`加一后存回ElasticSearch数据库。

## 6、GET操作：
指令示例
```
curl -v 10.29.2.1:12345/objects/test3
```
1、加入了元数据服务后，我们可以通过GET操作获取指定版本的对象，若不指定，则默认获取最新版本的对象。

2、本例中我们查询test3的最新版本，主要通过url的方式对ElasticSearch进行查询,
下面的语句表示查询名字为test3的版本最大的记录信息，并将元数据信息返回给接口服务节点。
```
"http://$(ES_SERVER)/metadata/_search?q=Name:test3&size=1&sort=Version:desc"
```
&emsp;&emsp;我们的主要目的是获取该对象对应版本的hash值及其分片id

&emsp;&emsp;下图中，我们主要模拟了有两个数据服务节点失效的情况，来演示数据服务节点的数据修复功能。

&emsp;&emsp;其中，1、2、4、6号数据服务节点数据完好，接口服务节点会向他们开启读通道。
2、5号节点数据损毁，接口服务节点回向他们开启写通道。

![c](./image/6-1.png)

&emsp;&emsp;接下来我们对2、5号节点上的数据进行修复。
![c](./image/6-2.png)
&emsp;&emsp;在上图中的第二步之后，我们就已经得到了修复后的对象数据，将该对象数据返回便是我们GET操作希望的结果。

&emsp;&emsp;在GET操作结束时，我们将调用commit方法将我们修复后存放在硬盘临时区域中的文件转为正式区域的文件。

## 7、Delete操作
删除操作主要是通过向ElasticSearch数据库输入特定值实现的。

1、首先向ElasticSearch查询对象最新的元数据信息

2、根据该元数据信息，将其版本`Version`加1，将其`hash`置为空字符串,将大小`Size`置为0，存放回ElasticSearch中，以此表示该对象已经被删除。



## 8、功能与不足

### 实现：

* 1、实现了接口服务与数据服务分离的架构，任意新主机只需要向`RabbitMQ`注册，
即可获取数据服务的支持。或是作为数据服务节点，承载数据服务的能力。
* 2、实现了元数据服务，完成了对于存储对象的版本控制。
* 3、实现了数据去重功能，避免了相同数据多次传输导致数据服务节点快速存满的问题
* 4、实现了数据校验功能，防止了不完整数据被存入数据节点
* 5、实现了数据冗余功能，将数据分片存储在多个数据服务节点上，且通过RS纠删码技术允许通过部分分片恢复整体数据。

### 不足：

1、每一次新对象上传时都需要进行去重操作（第五节的第3步），所以每一次PUT操作都需要等待数据服务节点的响应（这里我们设置的是1s）。
这也就导致，如果我们频繁的传输新的小对象，那么等待时间可能会远远大于传输时间。
一个有效的解决方法是将多个小对象组成一个大对象后上传，这样1s的时间也就微不足道了。

2、删除操作不会真的删除磁盘上的文件，这也就导致我们可以恢复自己删除的文件，但是磁盘的空间并不会因为删除操作而得到释放。
