# circle

1.各层可测试  全New注入
2.define和domain分开
3.各层日志属于各层
4.时间类型，全UTC
5.统一签名 请求 响应 error(服务器错误)
6.自动根据结构体生成openapi 带 go-playground标签
7.界面上可插件页面、后端数据可编程
8.Makefile管理
9.internal包的使用
10.目录服务器，只允许.app域名
11.超过32MB 使用流式
12.hashicorp/memberlist
13.DDNS
14.基于内存的虚拟文件系统，为了安全？？？
15.https://github.com/unrolled/secure 保证httpserver安全
16.https://github.com/cossacklabs/themis 端到端加密 零知识 面向会话的加密数据交换 分享加密
17.https://github.com/awnumar/memguard 防止数据接触磁盘
18.让读写磁盘无法人工篡改，无root权限类似。数据存储、传输、计算都加密无法人工干预，交换时使用零知识
19.分层：传输层包括底层（使用反射屏蔽和配置），拦截器层（待定），业务逻辑层组装层（Go 标准库），数据库操作层/抽象为接口（），CoreDomainDefine等
20.传输层都可以配置
21.匿名记录访客，且根据性能动态设置自己接受的访客个数
22.github.com/codenotary/immudb 不可变数据库
23.核心想解决的问题：隐私存储隐私计算隐私访问+数据分散计算可信
24.用户把数据上传至平台，平台方不能删除？？？
平台和用户之间要签订一些协议，数据所有权是归用户。不会存在恶意删号恶意处理的行为。
上传到这台计算机，平台不知道？？也不可删除，只有用户自己可以删除？？
比如 节点提供磁盘  却不知道  具体有哪些用户在自己这里上传，只知道自己的用户总数，自己的被访问热度？
也不知道自己的热门视频具体是哪些？用户知道这个服务器很热门，然后订阅。服务器只知道自己很热门，
但是不知道自己的因为什么而热门，然后这个服务器就可以接广告了。
为啥？比如用户拍了一个短视频，发快手、发抖音、发西瓜。
但是他们三个平台都不知道ID，无法删除和操作？？？，用户自己知道。用户知道哪个收益高。于是就多发某个平台。
有效杜绝了垄断？？？因为用户会自己投票去哪个平台？？？想走就走，所以平台需要拼命做好服务？？？
托管平台，发布平台，用户。托管的和平台各司其职。
托管的由政府保管，用户通过托管平台授予平台访问权。
核心技术 是托管平台 授权
核心需求 短视频联盟访问权 分散式节点计算
