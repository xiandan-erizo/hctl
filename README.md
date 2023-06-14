htl 自用工具

## 功能
检查deployment和cloneset的启动情况,启动完成就发送飞书通知

### 示例
```shell
# 检查deployment
htl status
# 检查cloneset
htl status -t clo
```
### 效果如下

![img.png](image/dep.png)

![img_1.png](image/message.png)
## 自定义config
配置目录 ~/.htl/config.yaml
```yaml
feishu:
  url: ''
  msg: "服务重启完成"
dump:
  ossRs: centos
  ossPod: "/javatmp/"
  ossPath: "/javatmp/"
```
## Feature

### 检测服务状态

- 多种资源类型要实时监控状态，报告异常的 pod
- 发送机器人消息，可以使用 xml 或者 yaml 定义机器人和发送消息文本

### 批量停止、启动服务

- 将副本数置为 0，并将副本数保存，方便下次拉起

### 支持 dump

- 执行命令

### 支持 SSH

- SSH node
