<img width="180" align="right" style="float: right; margin: 0 0 0 10px;" alt="Pomu Rainpuff" src="https://i.imgur.com/aH6F1Mh.png">

# pomu.app

pomu.app archives VTuber livestreams both automatically and on-demand.

[API documentation][7] / [CDN documentation][8]

# Requirements

pomu.app can be built and ran either as standalone or within
a Docker container.

**Building**

* Standalone
  * Go 1.18+
  * `node` 19+
  * `yarn`
  * `git`
* Docker
  * `docker`

**Running**

Only for standalone:

* `youtube-dl` / `yt-dlp`
* `ffmpeg`

For both standalone and within a Docker container:

* PostgreSQL database
* S3 object storage for finished files (we suggest [Backblaze][2])
* Google API key with YouTube v3 Data API access
* Discord OAuth application

Optional for both standalone and Docker:

* Sentry.io DSN for error reporting
* Holodex API key

# Building

**Backend (standalone)**

* Windows: `.\build.ps1`
* macOS and Linux: `./build.sh`

**Frontend (standalone)**

```
yarn install && yarn build
```

**Docker**

```
docker build .
```

# Running

First, rename the `.env.example` into `.env` and fill in your
configuration options.

**Standalone**

Starting pomu.app is as simple as running `pomu` or `pomu.exe`, depending on your OS.

**Docker**

> **Warning**  
> Do not change the `BIND_ADDRESS` value in `.env` when running pomu.app using Docker.

```
docker run <image> --name pomu -p 8080:8080 --env-file ".env"
```

You will receive a warning upon startup that the `.env` file was
not found by pomu. Docker has expanded the file already for us,
so this warning can be safely ignored.

# FAQ

Q: I'd like to take down an archived livestream on pomu.app   
A: Please contact mari@pomu.app or emily@pomu.app to initiate the take-down (DMCA) process.

Q: Why does this exist?  
A: We love watching various [Vtubers][0] which occasionally
do _unarchived_ livestreams, such as [karaoke][1]. We wanted to
archive them on a regular basis - thus pomu.app was born.  
  
Q: What does `pomu` stand for?  
A: [Pomu Rainpuff][3] is the [strongest fairy][4] in the world. It'd be only
fitting to have her as the name for this service.  
  
Q: How do you make money?  
A: We don't. The hosted version of pomu.app is fully paid
out of pocket by [Mari][5] and [Emily][6]. We don't intend to turn
it commercial as it's not our content, and thus we have no right
to profit off of livestreams produced by others.

Q: How much does it cost to run pomu.app per month?  
A: The server, paid for by [Emily][6], costs €5 per month. S3 storage, paid for
by [Mari][5], may reach up to €15 per month and are publicly displayed on [the development instance][9].

# Thank you

[![Instatus](https://avatars.githubusercontent.com/u/57594402?s=400&v=4)][instatus]

[Status page][10] provided by Instatus

[0]: https://en.wikipedia.org/wiki/VTuber
[1]: https://music.holodex.net/
[2]: https://www.backblaze.com/
[3]: https://www.youtube.com/channel/UCP4nMSTdwU1KqYWu3UH5DHQ
[4]: https://www.youtube.com/watch?v=iadFVBNQuMw
[5]: https://twitter.com/mellowagain
[6]: https://twitter.com/emilydotgg
[7]: https://docs.pomu.app
[8]: https://docs-cdn.pomu.app
[9]: https://dev.pomu.app
[10]: https://status.pomu.app
[instatus]: https://instatus.com/
