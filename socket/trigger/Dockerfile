FROM busybox

LABEL MAINTAINER Yoojia.Chen "yoojiachen@gmail.com"

# 执行文件名称，须与 name.txt 中一致
COPY edgex-trigger-socket /bin/

COPY application.toml /etc/edgex/

# 端口参考配置文件的通讯端口
EXPOSE 5570 5571

ENTRYPOINT ["edgex-trigger-socket"]