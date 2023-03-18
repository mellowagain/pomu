FROM node:19 AS frontend-builder

WORKDIR /usr/src/pomu
COPY . .

# install yarn
RUN corepack enable && \
    corepack prepare yarn@stable --activate

# build frontend files
RUN yarn install --immutable --immutable-cache && \
    yarn build

FROM golang:1.18 AS backend-builder

WORKDIR /usr/src/pomu
COPY . .

# build backend
RUN go build -v -ldflags "-X main.GitHash=${GITHUB_SHA}"

FROM ubuntu:jammy

# update apt mirrors
RUN apt-get update

# install runtime dependencies for youtube-dl
RUN apt-get install -y --no-install-recommends python3 curl ca-certificates
RUN update-alternatives --install /usr/bin/python python /usr/bin/python3 10 # set python3 as `python`

# direct pomu dependency: ffmpeg
RUN apt-get install -y --no-install-recommends ffmpeg
ENV FFMPEG="/usr/bin/ffmpeg"

# cleanup apt
RUN apt-get clean && \
    apt-get autoremove && \
    rm -rf /var/lib/apt/lists/*

# direct pomu dependency: yt-dlp
RUN curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux -o /usr/local/bin/yt-dlp
RUN chmod +x /usr/local/bin/yt-dlp
ENV YT_DLP="/usr/local/bin/yt-dlp"

WORKDIR /app/

COPY --from=frontend-builder /usr/src/pomu/dist /app/dist
COPY --from=backend-builder /usr/src/pomu/migrations /app/migrations
COPY --from=backend-builder /usr/src/pomu/pomu /app/

EXPOSE 8080
ENV BIND_ADDRESS="0.0.0.0:8080"
ENTRYPOINT ["/app/pomu"]
