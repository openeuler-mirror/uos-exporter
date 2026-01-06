
## 安装并配置 opengauss

```bash
yum install -y opengauss

cat >> /var/lib/opengauss/data/postgresql.conf <<"EOF"
listen_addresses = '*'
EOF


echo "host  all  all  0.0.0.0/0  sha256" >> /var/lib/opengauss/data/pg_hba.conf



systemctl restart opengauss
```

##  创建远程登录用户

```bash
su - opengauss
gsql -d postgres -p7654

ALTER ROLE "opengauss" PASSWORD "deepin12#$";

create user myuser with password 'deepin12#$' sysadmin ;
grant all PRIVILEGES to myuser; 
```


## 测试

```bash
gsql -dpostgres -h10.7.37.69 -Umyuser -r -W'deepin12#$'

gsql -d postgresql://myuser:"deepin12#$"@10.7.37.69:7654/postgres
```