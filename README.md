# ELAClient 节点控制及钱包客户端

## 安装
1. git clone http://git.elastos.org/elatest/ELAClient.git
2. cd ELAClient
3. make install
4. make

## 配置
项目目录下的cli-config.json文件。
配置文件不是必须的，没有配置文件时默认配置：LogToFile=false；IpAddress="localhost"；HttpJsonPort=20336。
当程序运行目录下有cli-config.json配置文件时，ela-cli程序将读取配置文件。
配置文件中的IpAddress和HttpJsonPort需正确配置，以连接到节点的JsonRPC端口。

## 功能列表
1. Wallet 钱包
    - 创建钱包 [--create, -c]，如：./ela-cli wallet -c
    - 钱包密码 [--password, -p]，如：./ela-cli wallet -c -p elatest，新建一个密码为elatest的钱包
    - 钱包名称 [--name, -n]，如：./ela-cli wallet -n my_wallet_name.dat -p elatest -c，新建一个文件名为my_wallet_name.dat，密码为elatest的钱包；注意，当使用多个参数时，-c必须放在最后。
    - 显示账户信息 [--account, -a]，如：./ela-cli wallet -a
    - 更改密码 --changepassword，如：./ela-cli wallet --changepassword
    - 重置钱包数据库 --reset，如：./ela-cli wallet --reset，已经同步的UTXO将被删除，已经添加的地址不会被删除
    - 增加账户 --addaccount，如：./ela-cli wallet --addaccount [public_key, public_key1,public_key2,...]，通过单个公钥添加标准账户，通过多个公钥添加多签账户
    - 删除账户 --deleteaccount，如：./ela-cli wallet --deleteaccount address，通过地址删除账户
    - 显示账户余额 [--balance, -b]，如：./ela-cli wallet -b，钱包会显示所有已经添加到钱包中的账户的余额
    - 交易 [--transacion, -t]，交易分为三种类型[create, sign, send]，可用参数：
      1. --from address，指定发起交易的账户地址
      2. --to address，指定接收账户的地址
      3. --amount number，指定发送的金额
      4. --fee number，指定交易的手续费
      5. --lock number，指定交易锁定到多少区块之后可以使用，高于当前区块高度才有作用
      6. --hex hex_string，指定被签名、发送的转账信息（hex string 格式）
      7. --file file_path，指定一个内容为[address, amount]格式的CSV文件创建多输出交易，或被签名、发送的转账文件

## 示例：
创建标准交易：
```
$ ./ela-cli wallet -t create --from 8JiMvfWKDwEeFNY3KN38PBif19ZhGGF9MH --to EXYPqZpQQk4muDrdXoRNJhCpoQtFBQetYg --amount 10000 --fee 0.00001
020001001335353737303036373931393437373739343130012f9931a59bd996bf7142eb49546df7f83946e27ab4999a69533949093ace5df305000000000002b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a30010a5d4e800000000000000219dd292acd80d7dd879be266b98183f1bba9adbabb037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3603071d5c7aa0b00000000001227cfba7dc38ff5a5a710b0c29109474c432d358b0000000000
```

创建多输出交易：
```
$ ./ela-cli wallet -t create --from 8JiMvfWKDwEeFNY3KN38PBif19ZhGGF9MH --file addresses.csv --fee 0.00001
2018/01/05 20:29:52.372532 [Multi output address: EY55SertfPSAiLxgYGQDUdxQW6eDZjbNbX , amount: 10000]
2018/01/05 20:29:52.372612 [Multi output address: Eeqn3kNwbnAsu1wnHNoSDbD8t8oq58pubN , amount: 10000]
2018/01/05 20:29:52.372630 [Multi output address: EXWWrRQxG2sH5U8wYD6jHizfGdDUzM4vGt , amount: 10000]
2018/01/05 20:29:52.372644 [Multi output address: ERjYb2WGUDUvfb2dDpK7Jygy1ZSyDGaAWL , amount: 10000]
2018/01/05 20:29:52.372662 [Multi output address: ENgeK613q1pPWWMNQL2ebA3G2Dwa92fTEN , amount: 10000]
020001001335353737303036373931393437373739343130014bd29e49527d7da1fc48684ebdc386efa9219568e1d97600fc6fc14d08a8754464000000000006b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a30010a5d4e80000000000000021a3a01cc2e25b0178010f5d7707930b7e41359d7eb037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a30010a5d4e80000000000000021ede51096266b26ca8695c5452d4d209760385c36b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a30010a5d4e800000000000000219d7798bd544db55cd40ed3d5ba55793be4db170ab037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a30010a5d4e800000000000000215e1db287d5ebdaa65e1d6418093bb5454dbf390ab037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a30010a5d4e800000000000000213ca8d9eddd4895f6156f12c2281f59f94c4a2ac4b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3484416aab0ab0b00000000001227cfba7dc38ff5a5a710b0c29109474c432d358b0000000000
```

签名交易：
```
$ ./ela-cli wallet -t sign --hex 020001001335353737303036373931393437373739343130012f9931a59bd996bf7142eb49546df7f83946e27ab4999a69533949093ace5df305000000000002b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a30010a5d4e800000000000000219dd292acd80d7dd879be266b98183f1bba9adbabb037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3603071d5c7aa0b00000000001227cfba7dc38ff5a5a710b0c29109474c432d358b0000000000
Password:
[ 1 / 2 ] Sign transaction successful
020001001335353737303036373931393437373739343130012f9931a59bd996bf7142eb49546df7f83946e27ab4999a69533949093ace5df305000000000002b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a30010a5d4e800000000000000219dd292acd80d7dd879be266b98183f1bba9adbabb037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3603071d5c7aa0b00000000001227cfba7dc38ff5a5a710b0c29109474c432d358b000000000142406430bcb52422ba1eafc404d82a51ddbc990a599020999dc35e519b80005b1d6112c007046a7238e72b46457b601f8c5aadd7b61b80d33d89a7b713c25e30145f00695221020c866d2cae5c886144086c05e2d124688bb4c8c289c6ebb928bbfc147ce0ebb421032f33edcbd864e89a7714b75f7c483ae4ca71e7c54b3901e47dff508914f1e0df2102af3ee9ae7de601a222fb7b5a94ba0d8ddb8a9db581438a04aa75fbf359cda03653ae
```

发送交易：
```
$ ./ela-cli wallet -t send --hex 020001001335353737303036373931393437373739343130012f9931a59bd996bf7142eb49546df7f83946e27ab4999a69533949093ace5df305000000000002b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a30010a5d4e800000000000000219dd292acd80d7dd879be266b98183f1bba9adbabb037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3603071d5c7aa0b00000000001227cfba7dc38ff5a5a710b0c29109474c432d358b000000000182406430bcb52422ba1eafc404d82a51ddbc990a599020999dc35e519b80005b1d6112c007046a7238e72b46457b601f8c5aadd7b61b80d33d89a7b713c25e30145f40ba20f7e2527c5a0dd77c5e18f4d85073a20e24d356d8ad8656ad16f4f327650f549566ae4b4369767a638be012594f82dd92a7f5a7f873ca496236a8eff705f6695221020c866d2cae5c886144086c05e2d124688bb4c8c289c6ebb928bbfc147ce0ebb421032f33edcbd864e89a7714b75f7c483ae4ca71e7c54b3901e47dff508914f1e0df2102af3ee9ae7de601a222fb7b5a94ba0d8ddb8a9db581438a04aa75fbf359cda03653ae
f39780a91241e457c954375eb4c771af958c096dedfab9787984e1e7c9fb167d
```

查看余额：
```
$  ./ela-cli wallet -b
Synchronize blocks: >
================================================================================
Address:      EXYPqZpQQk4muDrdXoRNJhCpoQtFBQetYg
ProgramHash:  abdb9aba1b3f18986b260079d87d0dd8ac92d29d21
Balance:      10000
================================================================================
Address:      8JiMvfWKDwEeFNY3KN38PBif19ZhGGF9MH
ProgramHash:  8b352d434c470991c2b000a7a5f58fc37dbacf2712
Balance:      32839999.99996000
================================================================================
```

2. mining 挖矿
  - 开启/关闭挖矿 [--toggle, -t] [start, stop]，如：./ela-cli mining -t start，开启挖矿
  - 手动挖矿 [--mine, -m] number，如：./ela-cli mining -m 100，产生100个区块

3. info 查询信息
  - 当前区块链上有多少区块 [--blockcount, --bc]，如：./ela-cli info --bc
  - 获取区块 [--getblock, --gb]，如：./ela-cli info --gb 100 或 ./ela-cli info --gb 92a257278a75864da0bd45171b190031719586a3984ba3363bb2575a1f620762
  - 获取交易 [--gettransacion, --gt]，如：./ela-cli info --gt 702ddbf789fc060bb844f018d0b0e107e6c67b4da1ca09c4e70cc40c19d0fccc
  - 获取最新区块的Hash --bestblockhash，如：./ela-cli info --bestblockhash
  - 获取节点连接数 [--connections, --cs]，如：./ela-cli info --cs
  - 获取邻居节点信息 [--neighbor, --nb]，如：./ela-cli info --nb
  - 获取节点状态信息 [--state, -s]，如：./ela-cli info -s
  - 获取节点版本 [--version, -v]，如：./ela-cli info -v

4. debug 设置日志输出级别[0~6]，0输出所有日志，6只输出错误日志
  - 设置日志输出级别 [--level, -l] number，如：./ela-cli debug -l 0，设置输出所有日志信息
