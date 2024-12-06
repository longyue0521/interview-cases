## 表锁案例

## 场景描述
通过id去查找记录，更新数据。

## 复现表锁
运行case30_test.go下的TestCase30函数。这个测试会去抢占表锁。类似如下sql
```shell
-- 记得先开启事务
-- 在这里拿到了 status = 0 而后执行一大堆的业务操作
SELECT * FROM case30 WHERE id = 20000 FOR UPDATE;  
UPDATE case30 SET status = 1 WHERE id = 20000;
```

执行如下sql查看是否表锁，

```sql
SELECT * FROM performance_schema.data_locks;
```

![img.png](img.png)

## 修复方案

将select for update 操作改成乐观锁操作， 也就是类似如下sql，
```shell
- 在这里拿到了 status = 0 一大堆的业务操作
SELECT * FROM case30 WHERE id = 20000; 
UPDATE case30 SET status = 1 WHERE id = 20000 AND status = 0;
```
修改后的代码可以参考case30_test.go下的TestCas这个函数