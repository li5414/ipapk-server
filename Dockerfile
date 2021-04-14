FROM alpine:latest
RUN mkdir -p /ipapk
COPY ipapk-server-v1.0.0-linux /ipapk/
COPY config.json /ipapk/
COPY public /ipapk/public
EXPOSE 8089
WORKDIR /ipapk
CMD ["./ipapk-server-v1.0.0-linux"]