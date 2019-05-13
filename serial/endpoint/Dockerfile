FROM busybox

LABEL MAINTAINER Yoojia.Chen "yoojiachen@gmail.com"

COPY edgex-endpoint.bin /app/
COPY application.toml /app/
WORKDIR /app/

# 端口参考配置文件的RPC通讯端口
EXPOSE 5573

ENTRYPOINT ["./edgex-endpoint.bin"]