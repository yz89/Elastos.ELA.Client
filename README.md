# ELAClient 节点控制及钱包客户端

## 功能列表
1. Wallet 钱包
    - 创建钱包 [--create, -c]，如：./ela-cli wallet -c
    - 钱包密码 [--password, -p]，如：./ela-cli wallet -c -p elatest，新建一个密码为elatest的钱包
    - 显示账户信息 [--account, -a]，如：./ela-cli wallet -a
    - 更改密码 --changepassword，如：./ela-cli wallet --changepassword
    - 重置钱包数据库 --reset，如：./ela-cli wallet --reset，已经同步的UTXO将被删除，已经添加的地址不会被删除
    - 增加账户 --addaccount，如：./ela-cli wallet --addaccount [public_key, public_key1,public_key2,...]，通过单个公钥添加标准账户，通过多个公钥添加多签账户
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
创建交易：
```
$ ./ela-cli wallet -t create --from 8JiMvfWKDwEeFNY3KN38PBif19ZhGGF9MH --to EXYPqZpQQk4muDrdXoRNJhCpoQtFBQetYg --amount 10000 --fee 0.00001
020001001335353737303036373931393437373739343130012f9931a59bd996bf7142eb49546df7f83946e27ab4999a69533949093ace5df305000000000002b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a30010a5d4e800000000000000219dd292acd80d7dd879be266b98183f1bba9adbabb037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3603071d5c7aa0b00000000001227cfba7dc38ff5a5a710b0c29109474c432d358b0000000000
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

4. debug 设置日志输出级别[0~6]，0输出所有日志，6只输出错误日志
