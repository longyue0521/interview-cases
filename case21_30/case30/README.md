### MetaDataLock案例

#### 模拟场景
    给一个热点表加一个字段。

#### 模拟出现MetaDataLock
运行case30_test.go下的测试用例TestMetaDataLock

#### 验证出现MetaDataLock
show processlist;
![img.png](img.png)

#### 解决方案
首先我们要避免使用长事务，事务不提交就会一直占用metadata_lock。如果长时间没有执行完成ddl，可以先把ddl线程，kill掉。或者长事务kill掉。
如果该表是一个热点表，kill的方法就不太可行了。kill完一个线程以后立马又会有。可以在执行alter语句的时候加上超时时间，在规定时间内执行完最好执行不完也不会阻塞后面的语句。然后自行决定是否需要重试。
可以执行case30_test.go下的测试用例TestMetaDataLockWithTimeout

