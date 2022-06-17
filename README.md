# CIRCLE

Header 放在Context里面，

路由权限基于casbin在路由上做

数据权限在Context里做

数据权限只支持到where in 类型的

需要具体带数据权限的变量实现某接口，根据jwt的角色和用户ID，
加上return 字段名以及该有的数据权限枚举值
然后调用预先注入的函数

若是List型的，过滤掉无权限的。
若是ID型的，过滤掉无权限的。
直接拦截修改Request变量。
