FROM busybox:1.31.0-uclibc

COPY run /cnab/app/run
RUN chmod +x /cnab/app/run

ENTRYPOINT [ "/cnab/app/run" ]