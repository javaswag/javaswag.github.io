(function () {

// --- Utils ---
const isMobile = window.matchMedia('(max-width: 480px)').matches;
window.addEventListener('load', () => {
  if (isMobile)
    document.body.classList.add('--mobile');
});

const lerp = (x, y, a) => x * (1 - a) + y * a;
const invlerp = (x, y, a) => clamp(0, 1, (a - x) / (y - x));
const clamp = (min, max, a) => Math.min(max, Math.max(min, a));
const rerange = (x1,  y1,  x2,  y2,  a) => lerp(x2, y2, invlerp(x1, y1, a));

function bindPreventAll (cb) {
  return function (e) {
    e.preventDefault();
    e.stopPropagation();
    e.stopImmediatePropagation();
    return cb(e);
  }
}

const LocalStorage = {
  _: 'PlayerUI_',
  get (key) { return JSON.parse(localStorage.getItem(this._ + key)); },
  set (key, value) { return localStorage.setItem(this._ + key, JSON.stringify(value)); },
};

function _createEl (tag, props, children) {
  let $el = tag === _createEl.fragment ? document.createDocumentFragment() : document.createElement(tag);
  if (props) {
    Object.keys(props).forEach((key) => {
      const val = props[key];
      if (typeof $el[key] === 'object' && $el[key] !== null)
        return Object.assign($el.style, val);
      $el[key] = val;
    });
  }
  if (children)
    children.forEach($child => $child && $el.append($child));
  return $el;
}
_createEl.fragment = Symbol();

function createEl (a0, a1, a2) { // overloading
  if (Array.isArray(a0)) // 1st arg is children
    return _createEl(_createEl.fragment, null, a0);
  if (typeof a0 === 'object') // 1st arg is props
    return _createEl('div', a0, a1);
  return _createEl(a0, a1, a2);
}

function formatTime (float) {
  const hours = Math.floor(float / 3600);
  const min = `${ Math.floor((float % 3600) / 60) }`.padStart(2, '0');
  const sec = `${ Math.floor(float % 60) }`.padStart(2, '0');
  const minSec = `${min}:${sec}`;
  if (hours)
    return `${hours}:${ minSec }`;
  return minSec;
}

// ---
class Waveform {
  player = null;
  scaleX = 1;

  constructor (opts/* { waveform, $el }*/) {
    const { player, $el, waveformPath, height, barsCount } = opts;
    Object.assign(this, { player, $el, waveformPath, height, barsCount });

    this.barsCount = window.matchMedia('(max-width: 780px)').matches ? 110 : 220;
    // this.barsCount = 220;

    this.$el.style.height = this.height;

    this.handlePointer = this.handlePointer.bind(this);

    this.fetchAnalyzeData(this.waveformPath).then(data => {
      // console.log('Fetched: ', this.duration, this.data);
      this.draw();
      const w = this.$el.offsetWidth;
      this.$el.style.width = `fit-content`;
      const inw = this.$el.offsetWidth;
      this.scaleX = w / inw;
      this.$el.style.transform = `scaleX(${ this.scaleX })`;

      let $playback = null;
      this.$el.appendChild(
        ($playback = createEl('button', {
          className: '__playback',
          onclick: bindPreventAll(() => this.player.togglePlay()),
          style: { transform: `scaleX(${ 1/this.scaleX })` },
        }, [
          createEl({ className: 'Icon --playback' }),
        ])),
      );
      $playback.addEventListener('pointerdown', e => e.stopPropagation());

      // this.scaleX = this.$el.parentNode.offsetWidth / this.$el.offsetWidth;
      // this.$el.style.transform = `scaleX(${ this.scaleX })`;
      // this.$el.style.width = `fit-content`;

      // this.pollAudio();
      this.listenPointer();
    });
  }

  async fetchAnalyzeData (waveformPath) {
    const buf = await fetch(waveformPath, { mode: 'no-cors'})
      .then(resp => resp.arrayBuffer());
    const data = new Int16Array(buf);
    this.data = data;
    // duration is stored in the end
    this.duration = data[ data.length - 1 ];
    return this.data;
  }

  listenPointer () {
    this.pointerPress = false;
    this.$el.addEventListener('pointerdown', this.handlePointer);
    this.$el.addEventListener('pointerup', this.handlePointer);
    this.$el.addEventListener('pointermove', this.handlePointer);
  }

  handlePointer (e) {
    if (e.type === 'pointerdown')
      this.pointerPress = true;

    if (e.type === 'pointerup') {
      this.pointerPress = false;
      return;
    }

    if (this.pointerPress) {
      const seek = (e.x - this.$el.offsetLeft) / this.$el.offsetWidth;
      const time = (seek / this.scaleX) * this.duration;
      this.player.seek(time);
      try {
        this.player.$audio.play();
      } catch (e) {}
    }
  }

  draw () {
    const SAMPLE_LEN = Math.round((this.data.length-1) / this.barsCount);
    function SampleInfo (avg) {
      return {
        // min: Infinity,
        max: -Infinity,
        avg,
      };
    }

    const sampleArr = Object.assign([], SampleInfo(0));
    for (let i = 0; i < this.barsCount; i++) {
      const sample = this.data.slice(i * SAMPLE_LEN, (i+1) * SAMPLE_LEN);

      const sampleInfo = SampleInfo(sample[0]);
      for (let i = 0; i < SAMPLE_LEN; i++) {
        const d = Math.abs(sample[i]);
        // sampleInfo.min = Math.min(d, sampleInfo.min);
        sampleInfo.max = Math.max(d, sampleInfo.max);
        sampleInfo.avg = (sampleInfo.avg + d) / 2;
      }

      // sampleArr.min = Math.min(sampleInfo.min, sampleArr.min);
      sampleArr.max = Math.max(sampleInfo.max, sampleArr.max);
      sampleArr.avg = (sampleInfo.avg, sampleArr.avg) / 2;

      sampleArr.push(sampleInfo);
    }

    const LIMIT_AMP = (sampleArr.max - sampleArr.avg) / 1.2;
    // sampleArr[0].max = .5;

    sampleArr.forEach((sampleInfo, i) => {
      let val = (sampleInfo.max + sampleInfo.avg) / 2;
      val /= LIMIT_AMP;
      val = Math.min(val, 1);
      // val = Math.pow(val, .5);

      const $col = this.createCol(val);
      // $col.addEventListener('mouseover', e => {
      //   this.$el.style.setProperty('--over-i', i);
      // }, false);
      this.$el.appendChild($col);
    });

    window.ANLZ = sampleArr;
  }

  createCol (amp) {
    // console.log(amp)
    const $col = document.createElement('div');
    $col.classList.add('Waveform__col');
    const _amp = Math.pow(amp, .5) * this.height;
    $col.style.height = `${ _amp }px`;
    $col.style.marginBottom = `-${ _amp / 4. }px`;
    return $col;
  }
}

// ---
class PlayerUI {
  ref = {};

  constructor (opts) {
    this.player = opts.player;
    this.title = opts.title;

    this.handlePointerMove = this.handlePointerMove.bind(this);
    this.handleSeeking = this.handleSeeking.bind(this);
    this.handleVoluming = this.handleVoluming.bind(this);
    this.handleSpeeding = this.handleSpeeding.bind(this);
    this.init();

    if (!isMobile) {
      window.addEventListener('pointerup', () => {
        this._seeking = false;
        this._voluming = false;
        this._speeding = false;

        window.removeEventListener('pointermove', this.handlePointerMove);
      });

      window.addEventListener('pointerdown', e => window.addEventListener('pointermove', this.handlePointerMove));
    }
  }

  handlePointerMove (e) {
    if (this._seeking) this.handleSeeking(e);
    else if (this._voluming) this.handleVoluming(e);
    else if (this._speeding) this.handleSpeeding(e);
  }

  init () {
    function btn (name, onclick, props) {
      return createEl('button', { className: '__' + name, name, onclick, ...props },
          [ createEl({ className: 'Icon --' + name }) ]);
    }

    const { ref } = this;

    function btnHoldRepeatEvents (cb, immediate = true) {
      let actionInterval = -1;
      return {
        onpointerdown (e) {
          if (immediate)
            cb(e);
          actionInterval = setInterval(() => { cb(e) }, 500);
        },
        onpointerup (e) { clearInterval(actionInterval) },
        onpointerleave (e) { clearInterval(actionInterval) },
      };
    }

    this.$el = createEl({ className: 'PlayerUI' },
    [
      createEl({ className: '__inner' },
      [
        btn('backward', () => this.player.jumpToNextCue(-1), { accessKey: ',' }),
        btn('rewind-backward', null, btnHoldRepeatEvents(() => this.player.rewindTime(-1))),
        btn('playback', () => this.player.togglePlay(), { accessKey: 'p' }),
        btn('rewind-forward', null, btnHoldRepeatEvents(() => this.player.rewindTime(1))),
        btn('forward', () => this.player.jumpToNextCue(+1), { accessKey: '.' }),

        ...(isMobile ? [
          (ref.cueTitle = createEl({ className: '__title', textContent: this.title })),
        ] : [
          (ref.time = createEl({ className: '__time', textContent: '0:00' })),
          (ref.seek = createEl({ className: '__seek',
            onclick: this.handleSeeking,
            onpointerdown: e => { this._seeking = true },
          }, [
            (ref.cueTitle = createEl({ className: '__cue-title', textContent: '' })),
          ])),
          (ref.duration = createEl({ className: '__duration', textContent: '4:20' })),

          btn('speed', () => this.player.setSpeed(1), { accessKey: 'l' }),
          (ref.speedSlider = createEl({ className: '__vert-slider __speed-slider Popup--passive',
            onclick: this.handleSpeeding,
            onpointerdown: e => { this._speeding = true },
          })),
          btn('volume', () => this.player.toggleMute(), { accessKey: 'm' }),
          (ref.volumeSlider = createEl({ className: '__vert-slider __volume-slider Popup--passive',
            onclick: this.handleVoluming,
            onpointerdown: e => { this._voluming = true },
          })),
        ]),

      ])
    ]);
    document.body.appendChild(this.$el);
  }

  handleSeeking (e) {
    const { seek: $el } = this.ref;
    const { left, width } = $el.getBoundingClientRect();
    this.player.seekProgress((e.x - left) / width);
    this.player.$audio.play();
  }

  handleVoluming (e) {
    const { volumeSlider: $el } = this.ref;
    const { top, height } = $el.getBoundingClientRect();
    const volume = 1 - Math.min(1, Math.max(0, (e.y - top) / height));
    this.player.setVolume(volume);
  }

  handleSpeeding (e) {
    const { speedSlider: $el } = this.ref;
    const { top, height } = $el.getBoundingClientRect();
    let speed = 1 - Math.min(1, Math.max(0, (e.y - top) / height));
    speed = Math.round(speed * 7) / 7; // 8 steps
    speed = lerp(.5, 1.75, speed); // rerange
    this.player.setSpeed(speed);
  }
}

class PodcastPlayer {
  waveform = null;
  ui = null;

  played = false;
  playing = false;
  cues = [];

  constructor(opts) {
    const { $audio, $waveform, waveformPath, height, title } = opts;
    Object.assign(this, { $audio });

    this.waveform = new Waveform({
      $el: $waveform, waveformPath, height,
      player: this,
    });

    this.handlePollAudio = this.handlePollAudio.bind(this);
    this.listenToAudioEvent();

    // setTimeout(() => {
      this.updateCues();
      this.ui = new PlayerUI({ player: this, title });
    // });
    // this.$audio.style.display = 'none';
    if (!isMobile)
      this.setVolume(LocalStorage.get('volume') || 1);

    window.PodcastPlayer = this;
  }

  listenToAudioEvent () {
    this.seekInterval = -1;
    this.$audio.addEventListener('play', e => {
      this.seekInterval = setInterval(this.handlePollAudio, 490);
      this.playing = true;
      this.played = true;
      this.handlePollAudio();
    });
    this.$audio.addEventListener('loadedmetadata', e => {
      this.$audio.style.display = 'none';
      this.handlePollAudio();
    });
    this.$audio.addEventListener('pause', e => {
      clearInterval(this.seekInterval);
      this.playing = false;
      this.handlePollAudio();
    });
    this.$audio.addEventListener('seeked', this.handlePollAudio);
  }

  getProgress (time) {
    return (Number.isFinite(time) ? time : this.$audio.currentTime) / this.waveform.duration;
  }

  handlePollAudio (_time) {
    // waveform handling
    const prog = this.getProgress(_time);
    [...this.waveform.$el.children].forEach(($col, i) => {
      const colProg = i / this.waveform.barsCount;
      $col.classList.toggle('--past', colProg < prog);
    });

    {
      const { seek, time, duration, cueTitle } = this.ui.ref;
      if (seek)
        seek.style.setProperty('--progress', `${ prog * 100 }%`);

      if (time)
        time.textContent = formatTime(this.$audio.currentTime);

      if (duration)
        duration.textContent = formatTime(this.$audio.duration);

      if (cueTitle && (this.played || this.$audio.currentTime > 0))
        cueTitle.textContent = this.getCue().title;
    }

    this.updateSpeedView();

    this.ui.$el.classList.toggle('--playing', this.playing);
    this.waveform.$el.classList.toggle('--playing', this.playing);
    this.ui.$el.classList.toggle('--muted', this.$audio.volume === 0);
  }

  seekProgress (prog) {
    this.seek(prog * this.$audio.duration);
  }

  togglePlay () {
    if (this.playing)
      this.$audio.pause();
    else
      this.$audio.play();
  }

  toggleMute () {
    this._muted = !this._muted;
    if (this._muted) {
      this._volumeBeforeMute = this.$audio.volume;
      this.$audio.volume = 0;
    }
    else {
      this.$audio.volume = this._volumeBeforeMute;
    }
    this.handlePollAudio();
  }

  seek (time) {
    time = Math.min(this.$audio.duration, Math.max(0, time));
    console.log('Seeking to', time);
    this.$audio.currentTime = time;
    this.handlePollAudio(time);
  }

  updateCues (cues) {
    let idx = 0;
    for (let $cue of document.querySelectorAll('.__cue')) {
      $cue.dataset.cueIdx = idx;
      this.cues.push({
        time: Number($cue.dataset.cueTime),
        title: $cue.parentNode.childNodes[1].textContent,
        idx: idx++,
      });
    }
  }

  getCueIdx (time = this.$audio.currentTime) {
    const i = this.cues.findIndex(cue => cue.time > time);
    return i === -1 ? 0 : i - 1;
  }
  getCue (time = this.$audio.currentTime) {
    return this.cues[this.getCueIdx(time)];
  }

  jumpToNextCue (dir) {
    const currentCue = this.getCue();
    if (dir < 0 && this.$audio.currentTime > currentCue.time + 3)
      dir = 0;
    const cue = this.cues[currentCue.idx + dir];
    if (cue != null)
      this.seek(cue.time);
    this.$audio.play();
  }

  rewindTime (dir) {
    this.$audio.currentTime += dir * 10;
  }

  setVolume (volume) {
    this.$audio.volume = volume;
    this.ui.ref.volumeSlider.style.setProperty('--value', volume);
    LocalStorage.set('volume', volume);
  }

  setSpeed (speed) {
    this.$audio.playbackRate = speed;
    this.updateSpeedView();
  }

  updateSpeedView () {
    const speed = this.$audio.playbackRate;
    if (this.ui.ref.speedSlider)
      this.ui.ref.speedSlider.style.setProperty('--value', invlerp(.5, 1.75, speed));
  }
}

window.PodcastPlayer = PodcastPlayer;

})();