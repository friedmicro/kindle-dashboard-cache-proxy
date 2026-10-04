FROM docker.friedmicro-lab.org/base
WORKDIR /src
COPY go.mod ./
COPY main.go ./
RUN CGO_ENABLED=0 go build -o /out/cache-proxy .

USER 65534:65534
EXPOSE 8090
ENTRYPOINT ["/out/cache-proxy"]
