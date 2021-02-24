# ChangeLog - rewardcalculator

## 1.2.6 - 2021-02-24
* Optimize QUERY message handling

## 1.2.5 - 2021-01-11
* Add START_BLOCK message (#53)
* Fix QUERY message handling (#54)

## 1.2.4 - 2020-12-16
* Fix bug
  * store calculatingBH as Header data not request data
    * request.BH is Uint.max and it causes problem when reloading DB(only case when RC dies while processing calculate message)

## 1.2.3 - 2020-12-04
* Fix bugs by reflections package(#48)

## 1.2.2 - 2020-06-19
* Add integration test (#43)
* Graceful shutdown (#46)

## 1.2.1 - 2020-01-03
* Fix rollback function bugs (#42)
* Modify log format (#40)


## 1.2.0 - 2019-12-06
* Add rollback function  (#33)
* Improve IPC protocol  (#38)
* Fix bugs
  * Beta2 reward calculation bug (#39)
