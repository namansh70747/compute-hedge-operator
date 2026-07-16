FROM golang:1.26 AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ENV CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH}
RUN go build -o /out/operator ./cmd/operator \
    && go build -o /out/mockocpi ./cmd/mockocpi \
    && go build -o /out/gpuexporter ./cmd/gpuexporter

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=build /out/operator /operator
COPY --from=build /out/mockocpi /mockocpi
COPY --from=build /out/gpuexporter /gpuexporter
USER 65532:65532
ENTRYPOINT ["/operator"]
