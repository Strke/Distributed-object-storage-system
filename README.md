# Distributed-object-storage-system

# 启动方式
#### 数据服务节点启动
```
export RABBITMQ_SERVER=amqp://test:test@172.17.0.2:5672
LISTEN_ADDRESS=10.29.1.1:12345 STORAGE_ROOT=/tmp/1 go run dataServer/dataServer.go
```
#### 接口服务节点启动
```
LISTEN_ADDRESS=10.29.2.2:12345 go run ApiServer/ApiServer.go
```

## 接口和数据存储分离的架构

![1](./image/1.png)

该架构分为三层：

**接口服务层**：提供了对外的REST接口

**`RabbitMQ`**：负责心跳包和消息的传输

​	--`ApiServer exchange`：用于心跳包的传输。所有**接口服务节点**绑定该`exchange`，所有发往该`exchange`的消息都会被转发给绑定它的所有消息队列

​	--`DataServer exchange`：用于定位消息的传输。所有**数据服务节点**绑定该exchange，用于接收接口服务的定位消息。

**数据服务层**：提供了数据的存储功能

### 对于心跳信息的处理：

#### 从接口服务（`ApiServer/heartbeat`）的角度看：

设置了一个`map[string]time.Time`类型的`dataServers`变量存储存活的数据服务节点。接收心跳信息刷新每个已注册数据服务节点信息的存活时间，移除超时的数据服务节点相关信息。

#### 从数据服务（`dataServer/heartbeat`）的角度看：

每隔5秒向`RabbitMQ`的`apiServers exchange`发送一次心跳信息，即向所有接口服务节点注册自己的存在。

### 对于PUT操作：

1、命令 `curl -v 10.29.2.2:12345/objects/test2 -XPUT -d"this is object test2"`，`10.29.2.2:12345`为接口服务节点。

2、通过心跳包程序（`ApiServer/heartbeat/apiserver.go`)中的`ChooseRandomDataServer()`函数选择`dataServer`变量里注册的一个数据服务节点，假设选定的为`10.29.1.1：12345`数据服务节点

3、替换HTTP包中的ip地址，即`10.29.2.2:12345 -> 10.29.1.1：12345`。此时数据包就会被发往`10.29.1.1：12345`数据服务节点。

4、接下来由数据服务节点来执行数据更新的操作。

### 对于GET操作：

1、命令`curl 10.29.2.1:12345/objects/test2`,`10.29.2.1`为接口服务节点。

2、首先进行定位，获取对象名`test2`，并向`DataServer exchange`群发`test2`名字。

3、数据服务节点自查本地内容，如果本地存在，通过定位程序（`dataServer/locate/dataserver.go`）中第25行的`Send`函数向消息的发送方返回本服务节点的监听地址。假设为`10.29.1.1：12345`

4、替换HTTP包中的ip地址，即`10.29.2.2:12345 -> 10.29.1.1：12345`。此时就会通过HTTP协议获取数据服务节点上的地址，并通过io流显示给用户。

### 优势：

1、实现了接口服务与数据服务分离的架构，任意新主机只需要向`RabbitMQ`注册，即可获取数据服务的支持。或是作为数据服务节点，承载数据服务的能力。

### 不足：

1、由于`PUT`时随机选择数据服务节点，这就导致多次`PUT`相同内容会导致不同的数据节点上存储相同的内容。这里需要**数据去重**。（如果做了去重，还需要进行容灾）

2、版本控制问题。现在的分布式存储系统无法做到对于同一份数据不同版本的切换，这里需要用到**元数据服务**。

