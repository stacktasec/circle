# CIRCLE

Header 放在Context里面，

路由权限基于接口实现返回值来做
实现 正选 反选 列表等 指定角色权限

数据权限在Context里做

数据权限只支持到where in 类型的

需要请求体实现某接口，根据jwt的角色和用户ID，
加上return 字段名以及该有的数据权限枚举值字符串
然后调用预先注入的函数

若是List All Enum型的，直接写业务逻辑即可
若是传入Enums型的，得到对应数据的，过滤掉无权限的Enum
若是Enum型的，过滤掉无权限的Enum，可直接404
直接拦截修改Request变量。
