FROM alpine:3.21
RUN apk add --no-cache ca-certificates
CMD ["sleep", "infinity"]
