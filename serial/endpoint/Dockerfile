FROM busybox

LABEL MAINTAINER="yoojiachen@gmail.com"

# 执行文件名称，须与 name.txt 中一致
COPY edgex-endpoint-serial /bin/

COPY application.toml /etc/edgex/

# 端口参考配置文件的通讯端口
EXPOSE 5570

# 串口设备
VOLUME ["/dev/ttyUSB0"]

ENTRYPOINT ["edgex-endpoint-serial"]