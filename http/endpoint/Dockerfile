ARG BUSYBOX_IMG=busybox
FROM $BUSYBOX_IMG

LABEL MAINTAINER="yoojiachen@gmail.com"

# 执行文件名称，须与 name.txt 中一致
COPY edgex-endpoint-http /bin/

ONBUILD COPY application.toml /etc/edgex/

# 端口参考配置文件的RPC通讯端口
ONBUILD EXPOSE 5570

ENTRYPOINT ["edgex-endpoint-http"]