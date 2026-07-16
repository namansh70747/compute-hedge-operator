FROM node:22 AS web
WORKDIR /web
COPY cmd/console/web/package.json cmd/console/web/package-lock.json ./
RUN npm ci
COPY cmd/console/web/ ./
RUN npm run build

FROM golang:1.26 AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Overwrite the committed placeholder dist with the freshly built SPA so the
# console binary embeds the real UI.
COPY --from=web /web/dist ./cmd/console/web/dist

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ENV CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH}
RUN go build -o /out/operator ./cmd/operator \
    && go build -o /out/mockocpi ./cmd/mockocpi \
    && go build -o /out/gpuexporter ./cmd/gpuexporter \
    && go build -o /out/console ./cmd/console

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=build /out/operator /operator
COPY --from=build /out/mockocpi /mockocpi
COPY --from=build /out/gpuexporter /gpuexporter
COPY --from=build /out/console /console
USER 65532:65532
ENTRYPOINT ["/operator"]
