FROM busybox

LABEL MAINTAINER Yoojia.Chen "yoojiachen@gmail.com"

COPY edgex-trigger.bin /app/
COPY application.toml /app/
WORKDIR /app/

# 端口参考配置文件的RPC通讯端口
EXPOSE 5570

ENTRYPOINT ["./edgex-trigger.bin"]