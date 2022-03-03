# dtm

http dtm saga(目前只测试了saga  tcc和xa暂未测试)

# 示例启动步骤

## 创建数据库 以及表结构

```sql
create
database if not exists dtm_barrier
/*!40100 DEFAULT CHARACTER SET utf8mb4 */
;
drop table if exists dtm_barrier.barrier;
create table if not exists dtm_barrier.barrier
(
    id bigint
(
    22
) PRIMARY KEY AUTO_INCREMENT,
    trans_type varchar
(
    45
) default '',
    gid varchar
(
    128
) default '',
    branch_id varchar
(
    128
) default '',
    op varchar
(
    45
) default '',
    barrier_id varchar
(
    45
) default '',
    reason varchar
(
    45
) default '' comment 'the branch type who insert this record',
    create_time datetime DEFAULT now
(
),
    update_time datetime DEFAULT now
(
),
    key
(
    create_time
),
    key
(
    update_time
),
    UNIQUE key
(
    gid,
    branch_id,
    op,
    barrier_id
)
    );
```

## 创建数据库 mall

# 执行

dtm.exe go run main1.go go run main.go 查看dtm_barrier.barrier表数据变化 以及mall表数据变化
