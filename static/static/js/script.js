//window.audioCtx = new (window.AudioContext || window.webkitAudioContext)();

var ws = null;
var ctrack = null;
var wsInterval = null;
var lyricsInterval = null;
class musicPlayer {
  constructor() {
    this.play = this.play.bind(this);
    this.skip = this.skip.bind(this);
    this.skipBtn = document.getElementById("skip");
    this.skipBtn.addEventListener("click", this.skip);
    this.playBtn = document.getElementById("play");
    this.playBtn.addEventListener("click", this.play);
    this.controlPanel = document.getElementById("control-panel");
    this.isPlaying = false;
  }
  skip() {
    ws.send(JSON.stringify({ op: 4 }));
  }
  play() {
    if (!this.isPlaying) {
      this.playBtn.classList.add("playing");
      window.player.src = `/audio`;
      window.player.muted = false;
      window.player.play();
    } else {
      this.playBtn.classList.remove("playing");
      window.player.muted = true;
    }

    this.isPlaying = !this.isPlaying;
  }
}

const newMusicplayer = new musicPlayer();

function enqueue() {
  q = document.getElementById("query").value.trim();
  if (ws === null) return;
  ws.send(JSON.stringify({ op: 3, query: q }));
}
function setTrack(track) {
  console.log(track);
  if (track === null) {
    return;
  }
  ctrack = track;
  document.getElementById("artist").innerText = ctrack.artist.name;
  document.getElementById("name").innerText = ctrack.title;
  window.player.src = `/audio`;
  setTimeout(lyricsControl, 0);
  //let artworkBox = document.getElementsByClassName("album-art")[0];
  //artworkBox.style.backgroundImage = `url(${ctrack.album.cover})`;
}
function setListeners(count) {
  document.getElementById("listeners").innerText = `Listeners: ${count}`;
}
function showSubBox() {
  subBox = document.getElementById("sub");
  subBox.classList.add("active");
}
function hideSubBox() {
  subBox = document.getElementById("sub");
  subBox.classList.remove("active");
}
function toggleSubBox() {
  subBox = document.getElementById("sub");
  Array.from(subBox.classList).find(function(element) {
    return element !== "active"
      ? subBox.classList.add("active")
      : subBox.classList.remove("active");
  });
}
function showLyricsBox() {
  lyricsBox = document.getElementById("lyrics");
  lyricsBox.classList.add("active");
}
function hideLyricsBox() {
  lyricsBox = document.getElementById("lyrics");
  lyricsBox.classList.remove("active");
}
function toggleLyricsBox() {
  lyricsBox = document.getElementById("lyrics");
  Array.from(lyricsBox.classList).find(function(element) {
    return element !== "active"
      ? lyricsBox.classList.add("active")
      : lyricsBox.classList.remove("active");
  });
}
function initWebSocket() {
  if (window.location.protocol == "http:") {
    ws = new WebSocket(`ws://${window.location.host}/status`);
  } else {
    ws = new WebSocket(`wss://${window.location.host}/status`);
  }
  ws.onerror = err => {
    console.log(err);
    ws.close();
  };
  ws.onopen = event => {
    console.log("[WS] opened");
    ws.send(JSON.stringify({ op: 1 }));
    wsInterval = setInterval(() => {
      ws.send(JSON.stringify({ op: 8 }));
    }, 30000);
  };
  ws.onclose = event => {
    console.log("[WS] closed");
    clearInterval(wsInterval);
    setTimeout(initWebSocket, 1000);
  };
  ws.onmessage = event => {
    console.log(event.data);
    var msg = JSON.parse(event.data);
    switch (msg.op) {
      case 1:
        setTrack(msg.track);
        break;
      case 3:
        var subBox = document.getElementById("sub");
        var artistBox = subBox.getElementsByClassName("artist")[0];
        var titleBox = subBox.getElementsByClassName("name")[0];
        artistBox.innerText = "";
        titleBox.innerText = "";
        if (!msg.success) {
          titleBox.innerText = msg.reason;
        } else {
          titleBox.innerText = msg.track.title;
          artistBox.innerText = msg.track.artist.name;
        }
        showSubBox();
        setTimeout(hideSubBox, 3000);
        document.getElementById("query").value = "";
        break;
      case 4:
        var subBox = document.getElementById("sub");
        var artistBox = subBox.getElementsByClassName("artist")[0];
        var titleBox = subBox.getElementsByClassName("name")[0];
        artistBox.innerText = "";
        titleBox.innerText = "";
        if (!msg.success) {
          titleBox.innerText = msg.reason;
        } else {
          titleBox.innerText = "Skipped!";
        }
        showSubBox()
        setTimeout(hideSubBox, 2000);
        document.getElementById("query").value = "";
        break;
      case 5:
        setListeners(msg.listeners);
        break;
      default:
        break;
    }
  };
}
const node = document.getElementsByClassName("query-track")[0];
node.addEventListener("keydown", function(event) {
  if (event.key === "Enter") {
    event.preventDefault();
    enqueue();
  }
});
window.onload = function() {
  this.player = document.getElementById("audio-player");
  this.initWebSocket();
};

function lyricsControl() {
  clearInterval(lyricsInterval);
  hideLyricsBox();
  var player = document.getElementById("audio-player");
  var lyricsBox = document.getElementById("lyrics");
  lyricsBox.getElementsByClassName("original")[0].innerText = "";
  lyricsBox.getElementsByClassName("translated")[0].innerText = "";
  if (ctrack.lyrics == null || ctrack.lyrics.lrc == null) {
    return;
  }
  showLyricsBox();
  let idx = 0;
  lyricsInterval = setInterval(() => {
    if (ctrack.lyrics.lrc[idx].time.total < player.currentTime) {
      lyricsBox.getElementsByClassName("original")[0].innerText =
        ctrack.lyrics.lrc[idx].text;
      lyricsBox.getElementsByClassName("translated")[0].innerText =
        ctrack.lyrics.lrc[idx].translated;
      idx++;
    }
    if (idx >= ctrack.lyrics.lrc.length) {
      hideLyricsBox();
      clearInterval(lyricsInterval);
    }
  }, 100);
}
