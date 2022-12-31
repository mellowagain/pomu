<img width="180" align="right" style="float: right; margin: 0 0 0 10px;" alt="Pomu Rainpuff" src="https://i.imgur.com/aH6F1Mh.png">

# pomu.app

Archive ongoing and future YouTube livestreams onto disk and S3 or automatically re-upload them to YouTube.

[Public API documentation][7]

# Requirements

Building:

* Go 1.18+

Running:

* `youtube-dl` / `yt-dlp`
  * `ffmpeg`
  * Decent amount of filesystem storage
  * S3 object storage for finished files (we suggest [Backblaze][2])
  * Sentry.io project for error reporting (optional)

In order to build the project, run:

* Windows: `.\build.ps1`
  * macOS and Linux: `./build.sh`

Afterwards, rename the `.env.example` into `.env` and fill in your
configuration options. Starting pomu.app is as simple as running `pomu`
or `pomu.exe`, depending on your OS.

# FAQ

Q: I'd like to take down an archived livestream on pomu.app   
A: Please contact mari@pomu.app to initiate the take-down (DMCA) process.

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
by [Mari][5], may reach up to €15 per month.

# Thank you

[![Instatus](https://avatars.githubusercontent.com/u/57594402?s=400&v=4)][instatus]

Status page provided by Instatus

[0]: https://en.wikipedia.org/wiki/VTuber
[1]: https://music.holodex.net/
[2]: https://www.backblaze.com/
[3]: https://www.youtube.com/channel/UCP4nMSTdwU1KqYWu3UH5DHQ
[4]: https://www.youtube.com/watch?v=iadFVBNQuMw
[5]: https://twitter.com/mellowagain
[6]: https://twitter.com/emilydotgg
[7]: https://pomu.stoplight.io/docs/pomu
[instatus]: https://instatus.com/
