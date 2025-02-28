(function () {

const MAX_SPEED = 1.75;
const MIN_SPEED = .5;

// --- Utils ---
const TOGGLE = Symbol('TOGGLE');
const IOS = /iP(hone|ad|od)/.test((navigator.userAgentData || navigator).platform);

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
          onclick: bindPreventAll(() => this.player.setPlaying(TOGGLE)),
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
  shortcuts = {
    'p' () { this.player.setPlaying(TOGGLE) },
    'm' () { this.player.setMuted(TOGGLE) },
    ',' () { this.player.rewindTime(-1) },
    '.' () { this.player.rewindTime(1) },
    '!>' () { this.player.jumpToNextCue(-1) },
    '!<' () { this.player.jumpToNextCue(1) },
    'l' () { this.player.setSpeed(this.player.$audio.playbackRate + .25) },
    'k' () { this.player.setSpeed(this.player.$audio.playbackRate - .25) },
  };

  constructor (opts) {
    this.player = opts.player;
    this.title = opts.title;

    this.handlePointerMove = this.handlePointerMove.bind(this);
    this.handleSeeking = this.handleSeeking.bind(this);
    this.handleVoluming = this.handleVoluming.bind(this);
    this.handleSpeeding = this.handleSpeeding.bind(this);
    this.init();

    // if (!isMobile) {
      window.addEventListener('pointerup', () => {
        this._seeking = false;
        this._voluming = false;
        this._speeding = false;

        window.removeEventListener('pointermove', this.handlePointerMove);
      });

      window.addEventListener('pointerdown', e => window.addEventListener('pointermove', this.handlePointerMove));
    // }

    const preventShortcutTags = ['INPUT', 'TEXTAREA'];
    window.addEventListener('keypress', e => {
      if (preventShortcutTags.includes(e.currentTarget.tagName))
        return;

      const shortcut = `${ e.shiftKey ? '!' : '' }${ e.key.toLowerCase() }`;
      const shortcutFn = this.shortcuts[shortcut];
      shortcutFn && shortcutFn.call(this, e);
    });
  }

  handlePointerMove (e) {
    if (this._seeking) this.handleSeeking(e);
    else if (this._voluming) this.handleVoluming(e);
    else if (this._speeding) this.handleSpeeding(e);
  }

  init () {
    function btn (name, onclick, props, children = []) {
      return createEl('button', { className: '__' + name, name, onclick, ...props },
          [ createEl({ className: 'Icon --' + name }), ...children ]);
    }

    const { ref } = this;

    function btnHoldRepeatEvents (cb, immediate = true) {
      let actionInterval = -1;
      let preventClick = false;
      function clear () {
        clearInterval(actionInterval);
        setTimeout(() => { preventClick = false }, .016);
      }
      return {
        onclick (e) {
          if (!preventClick)
            cb(e);
        },
        onpointerdown (e) {
          if (immediate)
            cb(e);
          preventClick = true;
          actionInterval = setInterval(() => { cb(e) }, 500);
        },
        onpointerup: clear,
        onpointerleave: clear,
      };
    }

    this.$el = createEl({ className: 'PlayerUI' },
    [
      createEl({ className: '__inner' },
      [
        // btn('backward', () => this.player.jumpToNextCue(-1), { accessKey: ',' }),
        btn('rewind-backward', null, {
          title: `Rewind forward\nShortcuts: [<], [shift + <] prev chapter \n[LONG TAP] to repeat rewind`,
          ...btnHoldRepeatEvents(() => this.player.rewindTime(-1)),
        }),

        btn('playback', () => { this.player.setPlaying(TOGGLE) }),

        btn('rewind-forward', null, {
          title: `Rewind backward\nShortcuts: [>], [shift + >] next chapter \n[LONG TAP] to repeat rewind`,
          accessKey: '.',
          ...btnHoldRepeatEvents(() => this.player.rewindTime(1)),
        }),
        // btn('forward', () => this.player.jumpToNextCue(+1), { accessKey: '.' }),

        ...(isMobile ? [
          (ref.cueTitle = createEl({ className: '__title', textContent: this.title })),
        ] : [
          (ref.time = createEl({ className: '__time', textContent: '0:00' })),
          (ref.seek = createEl({ className: '__seek',
            onclick: bindPreventAll(this.handleSeeking),
            onpointerdown: e => { this._seeking = true },
          }, [
            (ref.cueTitle = createEl({ className: '__cue-title', textContent: '' })),
          ])),
          (ref.duration = createEl({ className: '__duration', textContent: '4:20' })),



          btn('volume', () => this.player.setMuted(TOGGLE), {
            title: `Volume\nShortcut: [M]`,
          }),
          (ref.volumeSlider = createEl({ className: '__vert-slider __volume-slider Popup--passive',
            onclick: bindPreventAll(this.handleVoluming),
            onpointerdown: e => { this._voluming = true },
          })),
        ]).concat([
          btn('speed', () => this.player.setSpeed(1), {}, [
            (ref.speedSlider = createEl({ className: '__vert-slider __speed-slider Popup--passive',
              onclick: bindPreventAll(this.handleSpeeding),
              onpointerdown: e => { this._speeding = true },
            })),
          ]),
        ]),

      ])
    ]);
    document.body.appendChild(this.$el);
  }

  speedSliderVisibility (state) {
    if (state == null)
      return !this.ref.speedSlider.style.display;
    this.ref.speedSlider.style.display = state ? '' : 'none';
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
    speed = lerp(MIN_SPEED, MAX_SPEED, speed); // rerange
    this.player.setSpeed(speed);
  }
}

class Cues {
  player = null;
  cues = [];
  $cues = [];
  currentCue = null;

  constructor (opts) {
    const { player } = opts;
    Object.assign(this, { player });

    this._updateCues();
  }

  _updateCues () {
    this.$cues = [...document.querySelectorAll('.PlayerUI__cue')];

    let idx = 0;
    for (let $cue of this.$cues) {
      $cue.dataset.cueIdx = idx;
      this.cues.push({
        time: Number($cue.dataset.cueTime),
        title: $cue.parentNode.querySelector('.PlayerUI__cue-text').textContent,
        idx: idx++,
      });
    }
    this.cues.reverse();
  }

  updateView (time = this.player.$audio.currentTime) {
    const currentCue = this.getCue(time);
    let changeCurrentCue = false;
    const $prevCue = this.$cues.find($cue => $cue.classList.contains('--active'));
    if ($prevCue) {
      const changeCurrentCue = currentCue.idx === Number($prevCue.dataset.cueIdx);
      if (!changeCurrentCue) {
        $prevCue.classList.remove('--active');
      }
    }
    const $nextCue = document.querySelector('.PlayerUI__cue[data-cue-idx="' + currentCue.idx + '"]')
    $nextCue.classList.add('--active');
  }

  getCueIdx (time = this.player.$audio.currentTime) {
    const idx = this.cues.findIndex(cue => cue.time <= time);
    return idx === -1 ? 0 : idx;
  }
  getCue (time = this.player.$audio.currentTime) {
    const idx = this.getCueIdx(time);
    return this.cues[idx];
  }
}

// ---
class PodcastPlayer {
  waveform = null;
  ui = null;
  cues = null;
  episodeId = null;
  episodeTitle = 'Title';

  played = false;
  playing = false;

  constructor(opts) {
    const { $audio, $waveform, waveformPath, height, title: episodeTitle, episodeId } = opts;
    Object.assign(this, { $audio, episodeId: Number(episodeId), episodeTitle });

    this.waveform = new Waveform({
      $el: $waveform, waveformPath, height,
      player: this,
    });

    this.handlePollAudio = this.handlePollAudio.bind(this);

    this.cues = new Cues({ player: this });
    // setTimeout(() => {
      this.ui = new PlayerUI({ player: this, title: episodeTitle });
    // });
    // this.$audio.style.display = 'none';
    if (!isMobile)
      this.setVolume(LocalStorage.get('volume') || 1);
    // this.setSpeed(LocalStorage.get('speed') || 1);

    this.listenToAudioEvent();

    window.PodcastPlayer = this;
  }

  setSavedSeek () {
    if (this.episodeId != null) {
      let seek = LocalStorage.get('seek-' + this.episodeId);
      if (seek) {
        if (IOS && !this.played)
          this.handlePollAudio(seek);
        else
          this.seek(seek);
      }
    }
  }

  listenToAudioEvent () {
    this.seekInterval = -1;
    this.$audio.addEventListener('play', e => {
      this.seekInterval = setInterval(this.handlePollAudio, 490);
      this.playing = true;
      this.played = true;
      this.handlePollAudio();
    });

    if (this.$audio.readyState === 0)
      this.$audio.addEventListener('loadedmetadata', e => this.handleLoaded());
    else
      this.handleLoaded();

    this.$audio.addEventListener('pause', e => {
      clearInterval(this.seekInterval);
      this.playing = false;
      this.handlePollAudio();
    });
    this.$audio.addEventListener('seeked', e => this.handlePollAudio());
  }

  handleLoaded () {
    setTimeout(() => this.setSavedSeek(), 30);

    this.$audio.style.display = 'none';
    this.handlePollAudio();
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

    const currentCue = this.cues.getCue(_time);
    {
      const { seek, time, duration, cueTitle } = this.ui.ref;
      if (seek)
        seek.style.setProperty('--progress', `${ prog * 100 }%`);

      if (time)
        time.textContent = formatTime(this.$audio.currentTime);

      if (duration)
        duration.textContent = formatTime(this.$audio.duration);

      if (cueTitle) {
        if (this.played || this.$audio.currentTime > 0)
          cueTitle.textContent = currentCue.title;
        else
          cueTitle.textContent = this.episodeTitle;
      }
    }

    this.updateSpeedView();

    this.ui.$el.classList.toggle('--playing', this.playing);
    this.waveform.$el.classList.toggle('--playing', this.playing);
    this.ui.$el.classList.toggle('--muted', this.$audio.volume === 0);

    // cue highlighting
    this.cues.updateView(_time);
    if (this.$audio.currentTime) {
      LocalStorage.set('seek-' + this.episodeId, this.$audio.currentTime);
    }
  }

  seekProgress (prog) {
    this.seek(prog * this.$audio.duration);
  }

  setPlaying (state) {
    if (state === TOGGLE)
      this.playing = !this.playing;
    else this.playing = state;

    if (this.playing) {
      if (IOS && !this.played) { // fix currentTime bug IOS
        // const prevVolume = this.$audio.volume;
        this.$audio.volume = 0;

        // return new Promise((resolve) => {
        //   setTimeout(() => {
            const play = this.$audio.play();
            return play.then(e => {
              this.setSavedSeek();
              this.$audio.pause();
              return new Promise((resolve) => {
                setTimeout(() => resolve(this.$audio.play()), 30)
              });
            });
        //   }, 30);
        // });
      }
      else
        return this.$audio.play();
    }
    else {
      this.$audio.pause();
      return Promise.resolve(false);
    }
  }

  setMuted (state) {
    if (state === TOGGLE)
      this._muted = !this._muted;
    else this._muted = state;

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
    if (this.$audio.duration)
      time = clamp(0, this.$audio.duration, time);
    console.log('Seeking to', time);
    this.$audio.currentTime = time;
    this.handlePollAudio(time);
  }

  jumpToNextCue (dir) {
    const currentCue = this.cues.getCue();
    if (dir < 0 && this.$audio.currentTime > currentCue.time + 3)
      dir = 0;
    const cue = this.cues.cues[currentCue.idx + dir];
    if (cue != null)
      this.seek(cue.time);
    this.$audio.play();
  }

  rewindTime (dir) {
    this.$audio.currentTime += dir * 5;
  }

  setVolume (volume) {
    this.$audio.volume = Math.pow(volume, 1.6);
    this.ui.ref.volumeSlider.style.setProperty('--value', volume);
    LocalStorage.set('volume', volume);
  }

  setSpeed (speed) {
    this.$audio.playbackRate = clamp(MIN_SPEED, MAX_SPEED, speed);
    this.updateSpeedView();
    // LocalStorage.set('speed', speed);
  }

  updateSpeedView () {
    const speed = this.$audio.playbackRate;
    if (this.ui.ref.speedSlider)
      this.ui.ref.speedSlider.style.setProperty('--value', invlerp(MIN_SPEED, MAX_SPEED, speed));
  }
}

window.PodcastPlayer = PodcastPlayer;

})();