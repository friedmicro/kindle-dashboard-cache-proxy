FROM docker.friedmicro-lab.org/base AS build
WORKDIR /src
COPY go.mod ./
COPY main.go ./
RUN CGO_ENABLED=0 go build -o /opt/cache-proxy .

FROM docker.friedmicro-lab.org/base-slim
COPY --from=build /opt/cache-proxy /opt/cache-proxy

USER 65534:65534
EXPOSE 8090
ENTRYPOINT ["/opt/cache-proxy"]
