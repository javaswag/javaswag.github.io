class Waveform {
  constructor (opts/* { waveform, $el }*/) {
    Object.assign(this, opts);

    this.$el.style.height = this.height;

    this.fetchAnalyzeData(this.waveformPath).then(data => {
      console.log('Fetched: ', this.duration, this.waveform);
      this.draw();
      this.pollAudio();
      this.listenPointer();
    });
  }

  async fetchAnalyzeData (waveformPath) {
    const buf = await fetch(waveformPath, { mode: 'no-cors'})
      .then(resp => resp.arrayBuffer());
    const analyzeData = new Int16Array(buf);
    this.waveform = analyzeData;
    this.duration = analyzeData[ analyzeData.length - 1 ];
    return this.waveform;
  }

  listenPointer () {
    this.pointerPress = false;
    this.$el.addEventListener('pointerdown', this.handlePointer);
    this.$el.addEventListener('pointerup', this.handlePointer);
    this.$el.addEventListener('pointermove', this.handlePointer);
  }

  handlePointer = (e) => {
    if (e.type === 'pointerdown')
      this.pointerPress = true;

    if (e.type === 'pointerup') {
      this.pointerPress = false;
      return;
    }

    if (this.pointerPress) {
      const seek = (e.x - this.$el.offsetLeft) / this.$el.offsetWidth;
      const time = seek * this.duration;
      this.$audio.currentTime = time;
      this.handlePollAudio(time);
    }
  }

  pollAudio () {
    this.seekInterval = -1;
    this.$audio.addEventListener('play', e => {
      this.seekInterval = setInterval(this.handlePollAudio, 300);
    });
    this.$audio.addEventListener('pause', e => clearInterval(this.seekInterval));
    this.$audio.addEventListener('seeked', this.handlePollAudio);
  }

  handlePollAudio = (time) => {
    const prog = (Number.isInteger(time) ? time : this.$audio.currentTime) / this.duration;

    // console.time('Toggle "--past" class');
    [...this.$el.children].forEach(($col, i) => {
      const colProg = i / 220;
      $col.classList.toggle('--past', colProg < prog);
    });
    // console.timeEnd('Toggle "--past" class');
  }

  draw () {
    const SAMPLE_LEN = Math.round(this.waveform.length / this.barsCount);
    function SampleInfo (avg) {
      return {
        // min: Infinity,
        max: -Infinity,
        avg,
      };
    }
    const sampleArr = Object.assign([], SampleInfo(0));
    for (let i = 0; i < this.barsCount; i++) {
      const sample = this.waveform.slice(i * SAMPLE_LEN, (i+1) * SAMPLE_LEN);

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
    $col.style.height = `${ Math.pow(amp, .5) * this.height }px`;
    return $col;
  }
}

// const waveform = new Waveform({
//   $el: document.querySelector('.Waveform'),
//   waveformPath: CONF.waveformPath,
//   height: 70,
//   barsCount: 220,
//   $audio: document.querySelector('#audio'),
//   audioPath: CONF.audioPath,
// });
