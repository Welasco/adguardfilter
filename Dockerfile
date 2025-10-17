# Build the application from source
FROM golang:1.25 AS build-go

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /adguardfilter
#######################################################################

# Build fronend from source
FROM node:22-trixie AS build-frontend
WORKDIR /app
COPY . .
RUN cd /app/frontend-adguardfilter \
    && npm install \
    && npm run build
#######################################################################

# Deploy the application binary into a lean image
FROM alpine AS build-release

WORKDIR /app

COPY --from=build-go /adguardfilter /app/adguardfilter
COPY --from=build-go /app/docker/startup /opt/startup
COPY --from=build-frontend /app/frontend-adguardfilter/dist /app/public

RUN chmod 755 /opt/startup/init_container.sh

EXPOSE 3000

#USER nonroot:nonroot

ENTRYPOINT ["/opt/startup/init_container.sh"]