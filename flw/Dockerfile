FROM golang:1.17.7-alpine as builder
WORKDIR /app
COPY . . 
ENV GOPATH=/app
RUN cd src && go install


FROM golang:1.17.7-alpine
COPY --from=builder /app/bin/main /app/bin/main
WORKDIR /app

EXPOSE 8080
CMD [ "bin/main" ]
