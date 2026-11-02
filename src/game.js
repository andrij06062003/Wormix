'use strict';
// Bolt Barrage — an original turn-based artillery game with tiny robots.
(() => {
  // ---------- constants ----------
  const W = 1600, H = 900, WATER_Y = H - 55, G = 520, R = 10, STEP = 1 / 60;
  const WALK = 70, JUMP_VY = -270, JUMP_VX = 130, FALL_SAFE = 420;
  const TURN_TIME = 30, RETREAT_TIME = 3, CHARGE_TIME = 1.1, WIND_ACC = 140;
  const TEAMS = [
    { name: 'Blue Bolts', color: '#3d8bff', dark: '#1d4fa8', names: ['Sprocket', 'Gizmo', 'Widget'] },
    { name: 'Red Rivets', color: '#ff5a4e', dark: '#a8261d', names: ['Rivet', 'Clank', 'Torque'] },
  ];
  const WEAPONS = [
    { id: 'rocket', name: 'Rocket', key: '1' },
    { id: 'grenade', name: 'Grenade', key: '2' },
    { id: 'airstrike', name: 'Air Strike', key: '3' },
  ];

  const cv = document.getElementById('game');
  const ctx = cv.getContext('2d');
  cv.width = W; cv.height = H;
  function fit() {
    const s = Math.min(innerWidth / W, innerHeight / H);
    cv.style.width = W * s + 'px'; cv.style.height = H * s + 'px';
  }
  addEventListener('resize', fit); fit();

  const rand = (a, b) => a + Math.random() * (b - a);
  const clamp = (v, a, b) => Math.max(a, Math.min(b, v));
  function rr(c, x, y, w, h, r) {
    c.beginPath(); c.moveTo(x + r, y);
    c.arcTo(x + w, y, x + w, y + h, r); c.arcTo(x + w, y + h, x, y + h, r);
    c.arcTo(x, y + h, x, y, r); c.arcTo(x, y, x + w, y, r); c.closePath();
  }

  // ---------- sound ----------
  let actx = null;
  function sfx(type) {
    try {
      actx = actx || new (window.AudioContext || window.webkitAudioContext)();
      const t = actx.currentTime;
      if (type === 'boom') {
        const len = actx.sampleRate * 0.5, buf = actx.createBuffer(1, len, actx.sampleRate), d = buf.getChannelData(0);
        for (let i = 0; i < len; i++) d[i] = (Math.random() * 2 - 1) * Math.pow(1 - i / len, 2.5);
        const src = actx.createBufferSource(); src.buffer = buf;
        const f = actx.createBiquadFilter(); f.type = 'lowpass'; f.frequency.value = 700;
        const g = actx.createGain(); g.gain.value = 0.5;
        src.connect(f).connect(g).connect(actx.destination); src.start(t);
      } else {
        const o = actx.createOscillator(), g = actx.createGain();
        o.type = type === 'fire' ? 'sawtooth' : 'square';
        o.frequency.setValueAtTime(type === 'fire' ? 320 : 600, t);
        o.frequency.exponentialRampToValueAtTime(type === 'fire' ? 90 : 300, t + 0.2);
        g.gain.setValueAtTime(0.08, t); g.gain.exponentialRampToValueAtTime(0.001, t + 0.22);
        o.connect(g).connect(actx.destination); o.start(t); o.stop(t + 0.25);
      }
    } catch (e) { /* audio unavailable */ }
  }

  // ---------- terrain ----------
  const tcv = document.createElement('canvas'); tcv.width = W; tcv.height = H;
  const tctx = tcv.getContext('2d');
  const bgcv = document.createElement('canvas'); bgcv.width = W; bgcv.height = H;
  const bgctx = bgcv.getContext('2d');
  const mask = new Uint8Array(W * H);

  function solid(x, y) {
    x |= 0; y |= 0;
    if (x < 0 || x >= W || y < 0) return false;
    if (y >= H) return true;
    return mask[y * W + x] === 1;
  }

  function genBackground() {
    const g = bgctx.createLinearGradient(0, 0, 0, H);
    g.addColorStop(0, '#5aa8e6'); g.addColorStop(0.7, '#bfe3f7'); g.addColorStop(1, '#f3e7c9');
    bgctx.fillStyle = g; bgctx.fillRect(0, 0, W, H);
    const layers = [['#9cc3dc', 0.45, 60], ['#7fa9c4', 0.55, 45]];
    for (const [col, base, amp] of layers) {
      const p = rand(0, 6.28);
      bgctx.fillStyle = col; bgctx.beginPath(); bgctx.moveTo(0, H);
      for (let x = 0; x <= W; x += 8) bgctx.lineTo(x, H * base + Math.sin(x * 0.006 + p) * amp + Math.sin(x * 0.017 + p * 2) * amp * 0.3);
      bgctx.lineTo(W, H); bgctx.fill();
    }
  }

  function genTerrain() {
    tctx.globalCompositeOperation = 'source-over';
    tctx.clearRect(0, 0, W, H);
    const p = [0, 0, 0, 0].map(() => rand(0, 6.28));
    const hts = new Float32Array(W);
    for (let x = 0; x < W; x++) {
      let h = H * 0.58 + Math.sin(x * 0.004 + p[0]) * 90 + Math.sin(x * 0.011 + p[1]) * 45 +
        Math.sin(x * 0.027 + p[2]) * 18 + Math.sin(x * 0.0017 + p[3]) * 60;
      const e = Math.min(x, W - 1 - x);
      if (e < 120) h += (120 - e) * (120 - e) / 40;
      hts[x] = h;
    }
    const grad = tctx.createLinearGradient(0, H * 0.3, 0, H);
    grad.addColorStop(0, '#8a5a32'); grad.addColorStop(1, '#3d2414');
    tctx.fillStyle = grad;
    tctx.beginPath(); tctx.moveTo(0, H);
    for (let x = 0; x < W; x++) tctx.lineTo(x, hts[x]);
    tctx.lineTo(W, H); tctx.closePath(); tctx.fill();
    // floating islands
    const islands = [];
    for (let i = 0; i < 2; i++) {
      const cx = i === 0 ? rand(380, 700) : rand(900, 1220), cy = rand(210, 300), rx = rand(70, 110);
      islands.push([cx, cy, rx]);
      tctx.beginPath(); tctx.ellipse(cx, cy, rx, 26, 0, 0, Math.PI * 2); tctx.fill();
    }
    // texture
    tctx.globalCompositeOperation = 'source-atop';
    for (let i = 0; i < 4500; i++) {
      tctx.fillStyle = Math.random() < 0.7 ? `rgba(0,0,0,${rand(0.05, 0.2)})` : `rgba(255,220,160,${rand(0.05, 0.15)})`;
      tctx.fillRect(rand(0, W), rand(150, H), rand(2, 6), rand(2, 5));
    }
    // grass
    tctx.globalCompositeOperation = 'source-over';
    tctx.strokeStyle = '#4caf50'; tctx.lineWidth = 9; tctx.lineJoin = 'round';
    tctx.beginPath();
    for (let x = 0; x < W; x++) x ? tctx.lineTo(x, hts[x]) : tctx.moveTo(x, hts[x]);
    tctx.stroke();
    for (const [cx, cy, rx] of islands) {
      tctx.beginPath(); tctx.ellipse(cx, cy, rx, 26, 0, Math.PI * 1.05, Math.PI * 1.95); tctx.stroke();
    }
    const d = tctx.getImageData(0, 0, W, H).data;
    for (let i = 0; i < W * H; i++) mask[i] = d[i * 4 + 3] > 40 ? 1 : 0;
  }

  function carve(x, y, r) {
    tctx.save();
    tctx.globalCompositeOperation = 'destination-out';
    tctx.beginPath(); tctx.arc(x, y, r, 0, Math.PI * 2); tctx.fill();
    tctx.globalCompositeOperation = 'source-atop';
    tctx.strokeStyle = 'rgba(35,18,6,0.85)'; tctx.lineWidth = 7;
    tctx.beginPath(); tctx.arc(x, y, r + 2, 0, Math.PI * 2); tctx.stroke();
    tctx.restore();
    const r2 = r * r;
    for (let yy = Math.max(0, (y - r) | 0); yy <= Math.min(H - 1, y + r); yy++) {
      for (let xx = Math.max(0, (x - r) | 0); xx <= Math.min(W - 1, x + r); xx++) {
        if ((xx - x) ** 2 + (yy - y) ** 2 <= r2) mask[yy * W + xx] = 0;
      }
    }
  }

  // ---------- game state ----------
  let units = [], projs = [], parts = [], pops = [], clouds = [];
  let state = 'menu', cpuMode = true, turnTeam = 0, turnIdx = [-1, -1], active = null;
  let timer = 0, stateT = 0, settleT = 0, weapon = 0, charging = false, power = 0;
  let wind = 0, teamAmmo = [2, 2], ai = null, shake = 0, time = 0, winner = null;
  let banner = { text: '', t: 0 };
  const keys = {}, mouse = { x: W / 2, y: H / 2 };
  let buttons = [];

  for (let i = 0; i < 7; i++) clouds.push({ x: rand(0, W), y: rand(40, 260), s: rand(0.6, 1.4) });

  function newGame(cpu) {
    cpuMode = cpu;
    genBackground(); genTerrain();
    units = []; projs = []; parts = []; pops = [];
    teamAmmo = [2, 2]; turnIdx = [-1, -1]; winner = null;
    const n = 6, startTeam = Math.random() < 0.5 ? 0 : 1;
    for (let i = 0; i < n; i++) {
      const x = clamp(220 + i * (W - 440) / (n - 1) + rand(-40, 40), 180, W - 180) | 0;
      let y = 0; while (y < H && !solid(x, y)) y++;
      const team = (i + startTeam) % 2;
      units.push({
        x, y: y - R - 1, vx: 0, vy: 0, team, hp: 100, alive: true, grounded: false,
        facing: x < W / 2 ? 1 : -1, aim: 0.5, name: TEAMS[team].names[Math.floor(i / 2)],
      });
    }
    nextTurn(true);
  }

  function nextTurn(first) {
    turnTeam = first ? (Math.random() < 0.5 ? 0 : 1) : 1 - turnTeam;
    const list = units.filter(u => u.team === turnTeam);
    for (let k = 0; k < list.length; k++) {
      turnIdx[turnTeam] = (turnIdx[turnTeam] + 1) % list.length;
      if (list[turnIdx[turnTeam]].alive) break;
    }
    active = list[turnIdx[turnTeam]];
    wind = Math.round(rand(-1, 1) * 10) / 10;
    timer = TURN_TIME; state = 'aim'; stateT = 0;
    power = 0; charging = false; weapon = 0;
    ai = isCPU(turnTeam) ? { phase: 'think', t: 0 } : null;
    banner = { text: `${active.name}'s turn (${TEAMS[turnTeam].name})`, t: 2.2 };
  }
  const isCPU = t => cpuMode && t === 1;

  // ---------- physics ----------
  function circleHits(cx, cy) {
    for (let k = 0; k < 8; k++) {
      const a = k * Math.PI / 4;
      if (solid(cx + Math.cos(a) * R, cy + Math.sin(a) * R)) return true;
    }
    return solid(cx, cy);
  }

  function hurt(u, d) {
    if (!u.alive || u.hp <= 0 || d <= 0) return;
    u.hp = Math.max(0, u.hp - d);
    pops.push({ x: u.x, y: u.y - 30, text: '-' + d, color: TEAMS[u.team].color, t: 1.4 });
  }

  function updUnit(u, dt) {
    if (!u.alive) return;
    u.vy += G * dt;
    const steps = Math.max(1, Math.ceil(Math.max(Math.abs(u.vx), Math.abs(u.vy)) * dt / 2));
    u.grounded = false;
    for (let i = 0; i < steps; i++) {
      const sx = u.vx * dt / steps;
      if (sx) {
        const nx = u.x + sx;
        if (!circleHits(nx, u.y)) u.x = nx;
        else {
          let ok = false;
          for (let up = 1; up <= 3; up++) if (!circleHits(nx, u.y - up)) { u.x = nx; u.y -= up; ok = true; break; }
          if (!ok) u.vx *= -0.3;
        }
      }
      const ny = u.y + u.vy * dt / steps;
      if (circleHits(u.x, ny)) {
        if (u.vy > 0) {
          if (u.vy > FALL_SAFE) hurt(u, Math.round((u.vy - FALL_SAFE) * 0.12));
          u.grounded = true;
          u.vx *= 0.5; if (Math.abs(u.vx) < 15) u.vx = 0;
        }
        u.vy = 0;
      } else u.y = ny;
    }
    if (u.y > WATER_Y + R) {
      u.alive = false; u.hp = 0; splash(u.x);
      pops.push({ x: u.x, y: WATER_Y - 30, text: 'Splash!', color: '#fff', t: 1.6 });
      if (u === active && (state === 'aim' || state === 'retreat')) { state = 'settle'; charging = false; }
    }
  }

  function walk(u, dir, dt) {
    u.facing = dir;
    if (!u.grounded) return;
    const nx = u.x + dir * WALK * dt;
    for (let up = 0; up <= 6; up++) {
      if (!circleHits(nx, u.y - up)) {
        u.x = nx; u.y -= up;
        for (let d = 0; d < 6 && !circleHits(u.x, u.y + 1); d++) u.y += 1;
        return;
      }
    }
  }

  function hitUnit(x, y, owner, t) {
    for (const u of units) {
      if (u.alive && !(u === owner && t < 0.25) && (u.x - x) ** 2 + (u.y - y) ** 2 < (R + 3) ** 2) return true;
    }
    return false;
  }

  function updProj(p, dt) {
    p.t += dt; p.vy += G * dt;
    if (p.type !== 'grenade') p.vx += wind * WIND_ACC * (p.type === 'bomb' ? 0.5 : 1) * dt;
    const steps = Math.max(1, Math.ceil(Math.hypot(p.vx, p.vy) * dt / 2));
    for (let i = 0; i < steps; i++) {
      const nx = p.x + p.vx * dt / steps, ny = p.y + p.vy * dt / steps;
      if (p.type === 'grenade') {
        if (solid(nx, p.y)) p.vx *= -0.5; else p.x = nx;
        if (solid(p.x, ny)) { if (Math.abs(p.vy) > 80) sfx('tick'); p.vy *= -0.45; p.vx *= 0.85; } else p.y = ny;
      } else {
        p.x = nx; p.y = ny;
        if (solid(p.x, p.y) || hitUnit(p.x, p.y, p.owner, p.t)) {
          if (p.type === 'bomb') explode(p.x, p.y, 38, 28); else explode(p.x, p.y, 48, 45);
          p.dead = true; return;
        }
      }
    }
    if (p.type === 'grenade' && p.t >= 3) { explode(p.x, p.y, 58, 50); p.dead = true; return; }
    if (p.y > WATER_Y) { splash(p.x); p.dead = true; return; }
    if (p.x < -400 || p.x > W + 400) p.dead = true;
    if (p.type !== 'grenade' && Math.random() < 0.8) {
      parts.push({ x: p.x, y: p.y, vx: rand(-20, 20), vy: rand(-20, 20), life: 0.6, max: 0.6, color: 'rgba(220,220,220,', size: rand(3, 6), grav: -30 });
    }
  }

  function explode(x, y, r, dmg) {
    carve(x, y, r);
    sfx('boom');
    for (const u of units) {
      if (!u.alive) continue;
      const dx = u.x - x, dy = u.y - y, d = Math.hypot(dx, dy), reach = r + R + 6;
      if (d < reach) {
        const f = 1 - d / reach, n = d || 1;
        hurt(u, Math.round(dmg * (0.35 + 0.65 * f)));
        u.vx += dx / n * f * 380; u.vy += dy / n * f * 380 - 160 * f; u.grounded = false;
      }
    }
    for (const p of projs) {
      if (p.type !== 'grenade' || p.dead) continue;
      const dx = p.x - x, dy = p.y - y, d = Math.hypot(dx, dy);
      if (d < r * 2) { const f = 1 - d / (r * 2), n = d || 1; p.vx += dx / n * f * 400; p.vy += dy / n * f * 400; }
    }
    for (let i = 0; i < 34; i++) {
      const a = rand(0, Math.PI * 2), s = rand(40, 260);
      parts.push({ x, y, vx: Math.cos(a) * s, vy: Math.sin(a) * s, life: rand(0.3, 0.7), max: 0.7,
        color: ['rgba(255,220,80,', 'rgba(255,140,40,', 'rgba(255,80,30,'][i % 3], size: rand(4, 9), grav: 0 });
    }
    for (let i = 0; i < 16; i++) {
      const a = rand(Math.PI, Math.PI * 2), s = rand(100, 300);
      parts.push({ x, y, vx: Math.cos(a) * s, vy: Math.sin(a) * s, life: rand(0.6, 1.2), max: 1.2,
        color: 'rgba(110,70,40,', size: rand(2, 5), grav: G });
    }
    for (let i = 0; i < 10; i++) {
      parts.push({ x: x + rand(-r / 2, r / 2), y: y + rand(-r / 2, r / 2), vx: rand(-20, 20), vy: rand(-60, -20),
        life: rand(0.8, 1.6), max: 1.6, color: 'rgba(80,80,80,', size: rand(10, 18), grav: -20 });
    }
    shake = Math.min(20, shake + r / 4);
  }

  function splash(x) {
    for (let i = 0; i < 20; i++) {
      parts.push({ x: x + rand(-8, 8), y: WATER_Y, vx: rand(-80, 80), vy: rand(-280, -100), life: 0.9, max: 0.9,
        color: 'rgba(180,220,255,', size: rand(2, 5), grav: G });
    }
  }

  // ---------- actions ----------
  function fire(targetX) {
    const u = active, w = WEAPONS[weapon];
    if (w.id === 'airstrike') {
      if (teamAmmo[u.team] <= 0) return;
      teamAmmo[u.team]--;
      for (let i = 0; i < 5; i++) {
        projs.push({ type: 'bomb', x: targetX - 80 + i * 40 - wind * 60, y: -30 - i * 25, vx: wind * 40, vy: 140, t: 0, owner: null });
      }
    } else {
      const dx = u.facing * Math.cos(u.aim), dy = -Math.sin(u.aim), sp = 250 + power * 650;
      projs.push({ type: w.id, x: u.x + dx * 16, y: u.y - 2 + dy * 16, vx: dx * sp, vy: dy * sp, t: 0, owner: u });
    }
    sfx('fire');
    charging = false; power = 0; state = 'fired'; stateT = 0;
  }

  function jump() {
    const u = active;
    if (u && u.alive && u.grounded) { u.vy = JUMP_VY; u.vx = u.facing * JUMP_VX; u.grounded = false; }
  }

  // ---------- CPU opponent ----------
  function simRocket(x, y, vx, vy, owner) {
    let t = 0;
    for (let f = 0; f < 480; f++) {
      t += STEP; vy += G * STEP; vx += wind * WIND_ACC * STEP;
      const steps = Math.max(1, Math.ceil(Math.hypot(vx, vy) * STEP / 2));
      for (let i = 0; i < steps; i++) {
        x += vx * STEP / steps; y += vy * STEP / steps;
        if (solid(x, y) || hitUnit(x, y, owner, t)) return { x, y };
      }
      if (y > WATER_Y || x < -400 || x > W + 400) return null;
    }
    return null;
  }

  function planShot() {
    const u = active;
    const enemies = units.filter(e => e.alive && e.team !== u.team && e.hp > 0);
    let best = { score: Infinity, angle: 0.6, facing: u.facing, power: 0.6 };
    for (const facing of [-1, 1]) {
      for (let a = -1.2; a <= 1.45; a += 0.07) {
        for (let pw = 0.1; pw <= 1.0001; pw += 0.05) {
          const dx = facing * Math.cos(a), dy = -Math.sin(a), sp = 250 + pw * 650;
          const hit = simRocket(u.x + dx * 16, u.y - 2 + dy * 16, dx * sp, dy * sp, u);
          if (!hit) continue;
          let score = Infinity;
          for (const e of enemies) score = Math.min(score, Math.hypot(e.x - hit.x, e.y - hit.y));
          for (const f of units) {
            if (f.alive && f.team === u.team) { const d = Math.hypot(f.x - hit.x, f.y - hit.y); if (d < 75) score += (75 - d) * 4; }
          }
          if (score < best.score) best = { score, angle: a, facing, power: pw };
        }
      }
    }
    best.angle = clamp(best.angle + rand(-0.04, 0.04), -1.45, 1.45);
    best.power = clamp(best.power + rand(-0.03, 0.03), 0.05, 1);
    Object.assign(ai, best);
  }

  function aiUpdate(dt) {
    const u = active;
    ai.t += dt;
    if (ai.phase === 'think') { if (ai.t > 0.8) { planShot(); ai.phase = 'aim'; } return; }
    if (ai.phase === 'aim') {
      u.facing = ai.facing;
      const d = ai.angle - u.aim, st = 1.4 * dt;
      if (Math.abs(d) <= st) { u.aim = ai.angle; ai.phase = 'charge'; charging = true; power = 0; }
      else u.aim += Math.sign(d) * st;
      return;
    }
    if (ai.phase === 'charge') {
      power += dt / CHARGE_TIME;
      if (power >= ai.power) { power = ai.power; fire(); }
    }
  }

  // ---------- update ----------
  function update(dt) {
    time += dt;
    for (const c of clouds) { c.x += (8 + wind * 40) * c.s * dt; if (c.x > W + 150) c.x = -150; if (c.x < -150) c.x = W + 150; }
    if (state !== 'menu') {
      for (const u of units) updUnit(u, dt);
      for (const p of projs) if (!p.dead) updProj(p, dt);
      projs = projs.filter(p => !p.dead);
    }
    for (const p of parts) { p.life -= dt; p.vy += p.grav * dt; p.x += p.vx * dt; p.y += p.vy * dt; p.vx *= 0.98; }
    parts = parts.filter(p => p.life > 0);
    for (const p of pops) { p.t -= dt; p.y -= 25 * dt; }
    pops = pops.filter(p => p.t > 0);
    shake *= 0.88; banner.t -= dt;
    stateT += dt;

    const u = active;
    switch (state) {
      case 'aim': {
        timer -= dt;
        if (timer <= 0 || !u.alive || u.hp <= 0) { state = 'settle'; charging = false; break; }
        if (ai) { aiUpdate(dt); break; }
        if (!charging) {
          if (keys.ArrowLeft) walk(u, -1, dt);
          else if (keys.ArrowRight) walk(u, 1, dt);
        }
        if (keys.ArrowUp) u.aim = clamp(u.aim + 1.6 * dt, -1.45, 1.45);
        if (keys.ArrowDown) u.aim = clamp(u.aim - 1.6 * dt, -1.45, 1.45);
        if (charging) { power = Math.min(1, power + dt / CHARGE_TIME); if (power >= 1) fire(); }
        break;
      }
      case 'fired':
        if (projs.length === 0 && stateT > 0.3) {
          if (u.alive && !ai) { state = 'retreat'; timer = RETREAT_TIME; } else state = 'settle';
          stateT = 0;
        }
        break;
      case 'retreat':
        timer -= dt;
        if (keys.ArrowLeft) walk(u, -1, dt); else if (keys.ArrowRight) walk(u, 1, dt);
        if (timer <= 0 || !u.alive) { state = 'settle'; stateT = 0; }
        break;
      case 'settle':
        if (projs.length === 0 && units.every(v => !v.alive || v.grounded)) {
          const dying = units.find(v => v.alive && v.hp <= 0);
          if (dying) {
            dying.alive = false;
            pops.push({ x: dying.x, y: dying.y - 30, text: 'KO!', color: '#fff', t: 1.6 });
            explode(dying.x, dying.y, 30, 18);
            settleT = 0; break;
          }
          settleT += dt;
          if (settleT > 0.6) {
            settleT = 0;
            const alive = [0, 1].map(t => units.some(v => v.alive && v.team === t));
            if (!alive[0] || !alive[1]) { state = 'over'; winner = alive[0] ? 0 : alive[1] ? 1 : -1; }
            else nextTurn(false);
          }
        } else settleT = 0;
        break;
    }
  }

  // ---------- drawing ----------
  function drawBot(u, isActive) {
    const team = TEAMS[u.team];
    ctx.save();
    ctx.translate(u.x, u.y);
    ctx.scale(u.facing, 1);
    // treads
    ctx.fillStyle = '#2b2b33'; rr(ctx, -10, 3, 20, 7, 3.5); ctx.fill();
    ctx.fillStyle = '#666';
    for (const wx of [-6, 0, 6]) { ctx.beginPath(); ctx.arc(wx, 6.5, 2, 0, Math.PI * 2); ctx.fill(); }
    // body
    ctx.fillStyle = team.color; ctx.strokeStyle = team.dark; ctx.lineWidth = 2;
    rr(ctx, -8, -10, 16, 14, 4); ctx.fill(); ctx.stroke();
    // antenna
    ctx.strokeStyle = '#444'; ctx.lineWidth = 1.5;
    ctx.beginPath(); ctx.moveTo(-2, -10); ctx.lineTo(-4, -16); ctx.stroke();
    ctx.fillStyle = isActive ? (Math.sin(time * 8) > 0 ? '#ffe14d' : '#ff9f1a') : '#ccc';
    ctx.beginPath(); ctx.arc(-4, -17, 2.2, 0, Math.PI * 2); ctx.fill();
    // visor + eye
    ctx.fillStyle = '#14161c'; rr(ctx, -1, -8, 9, 6, 2); ctx.fill();
    ctx.fillStyle = '#7ff5ff'; ctx.beginPath(); ctx.arc(4.5, -5, 1.8, 0, Math.PI * 2); ctx.fill();
    // arm / launcher
    ctx.translate(1, -1); ctx.rotate(-(isActive ? u.aim : -0.4));
    ctx.fillStyle = '#555'; rr(ctx, 0, -2, isActive ? 13 : 8, 4, 2); ctx.fill();
    ctx.restore();

    ctx.font = 'bold 13px system-ui, sans-serif'; ctx.textAlign = 'center';
    ctx.lineWidth = 3; ctx.strokeStyle = 'rgba(0,0,0,.55)';
    ctx.strokeText(u.name, u.x, u.y - 36); ctx.fillStyle = '#fff'; ctx.fillText(u.name, u.x, u.y - 36);
    ctx.strokeText(u.hp, u.x, u.y - 22); ctx.fillStyle = team.color; ctx.fillText(u.hp, u.x, u.y - 22);
  }

  function drawProj(p) {
    ctx.save(); ctx.translate(p.x, p.y);
    if (p.type === 'grenade') {
      ctx.fillStyle = '#3c6e2f'; ctx.beginPath(); ctx.arc(0, 0, 5, 0, Math.PI * 2); ctx.fill();
      ctx.fillStyle = '#ddd'; ctx.fillRect(-1.5, -8, 3, 3);
      ctx.font = 'bold 14px system-ui'; ctx.textAlign = 'center'; ctx.fillStyle = '#fff';
      ctx.fillText(Math.max(1, Math.ceil(3 - p.t)), 0, -12);
    } else {
      ctx.rotate(Math.atan2(p.vy, p.vx));
      ctx.fillStyle = p.type === 'bomb' ? '#333' : '#d33'; rr(ctx, -8, -3, 14, 6, 3); ctx.fill();
      ctx.fillStyle = '#ffb020'; ctx.beginPath(); ctx.moveTo(-8, -2); ctx.lineTo(-14 - Math.random() * 4, 0); ctx.lineTo(-8, 2); ctx.fill();
    }
    ctx.restore();
  }

  function drawClouds() {
    ctx.fillStyle = 'rgba(255,255,255,.85)';
    for (const c of clouds) {
      ctx.beginPath();
      ctx.arc(c.x, c.y, 26 * c.s, 0, Math.PI * 2); ctx.arc(c.x + 28 * c.s, c.y - 10 * c.s, 30 * c.s, 0, Math.PI * 2);
      ctx.arc(c.x + 58 * c.s, c.y, 24 * c.s, 0, Math.PI * 2); ctx.fill();
    }
  }

  function drawWater() {
    for (let layer = 0; layer < 2; layer++) {
      ctx.fillStyle = layer ? 'rgba(25,80,160,.9)' : 'rgba(70,140,220,.6)';
      ctx.beginPath(); ctx.moveTo(0, H);
      for (let x = 0; x <= W; x += 10) ctx.lineTo(x, WATER_Y + layer * 8 + Math.sin(x * 0.02 + time * (2 + layer)) * 4);
      ctx.lineTo(W, H); ctx.fill();
    }
  }

  function drawAim() {
    const u = active;
    if (!u || !u.alive || state !== 'aim') return;
    ctx.save();
    const by = u.y - 46 + Math.sin(time * 5) * 4;
    ctx.fillStyle = TEAMS[u.team].color; ctx.strokeStyle = '#fff'; ctx.lineWidth = 2;
    ctx.beginPath(); ctx.moveTo(u.x - 8, by - 10); ctx.lineTo(u.x + 8, by - 10); ctx.lineTo(u.x, by); ctx.closePath(); ctx.fill(); ctx.stroke();
    if (WEAPONS[weapon].id === 'airstrike') {
      if (!ai) {
        ctx.setLineDash([8, 8]); ctx.strokeStyle = 'rgba(255,60,60,.8)';
        ctx.beginPath(); ctx.moveTo(mouse.x, 0); ctx.lineTo(mouse.x, mouse.y); ctx.stroke(); ctx.setLineDash([]);
        ctx.beginPath(); ctx.arc(mouse.x, mouse.y, 14, 0, Math.PI * 2); ctx.stroke();
      }
    } else {
      const dx = u.facing * Math.cos(u.aim), dy = -Math.sin(u.aim);
      const cx = u.x + dx * 64, cy = u.y - 2 + dy * 64;
      ctx.strokeStyle = '#fff'; ctx.lineWidth = 2;
      ctx.beginPath(); ctx.arc(cx, cy, 8, 0, Math.PI * 2); ctx.moveTo(cx - 12, cy); ctx.lineTo(cx + 12, cy);
      ctx.moveTo(cx, cy - 12); ctx.lineTo(cx, cy + 12); ctx.stroke();
      if (charging) {
        const n = Math.floor(power * 12);
        for (let i = 0; i <= n; i++) {
          const k = i / 12, d = 18 + k * 70;
          ctx.fillStyle = `hsl(${60 - k * 60},100%,55%)`;
          ctx.beginPath(); ctx.arc(u.x + dx * d, u.y - 2 + dy * d, 2 + k * 5, 0, Math.PI * 2); ctx.fill();
        }
      }
    }
    ctx.restore();
  }

  function text(t, x, y, size, color, align = 'center') {
    ctx.font = `bold ${size}px system-ui, sans-serif`; ctx.textAlign = align;
    ctx.lineWidth = Math.max(3, size / 6); ctx.strokeStyle = 'rgba(0,0,0,.6)'; ctx.strokeText(t, x, y);
    ctx.fillStyle = color; ctx.fillText(t, x, y);
  }

  function drawHUD() {
    // team panels
    for (let t = 0; t < 2; t++) {
      const x = t ? W - 290 : 20, team = TEAMS[t];
      const hp = units.filter(u => u.team === t && u.alive).reduce((s, u) => s + u.hp, 0);
      ctx.fillStyle = 'rgba(10,15,30,.65)'; rr(ctx, x, 16, 270, 58, 10); ctx.fill();
      text(team.name + (isCPU(t) ? ' (CPU)' : ''), x + 14, 40, 18, team.color, 'left');
      ctx.fillStyle = 'rgba(255,255,255,.15)'; rr(ctx, x + 14, 50, 242, 12, 6); ctx.fill();
      ctx.fillStyle = team.color; rr(ctx, x + 14, 50, Math.max(0, 242 * hp / 300), 12, 6); ctx.fill();
      if (turnTeam === t && state !== 'over') { ctx.strokeStyle = '#fff'; ctx.lineWidth = 2; rr(ctx, x, 16, 270, 58, 10); ctx.stroke(); }
    }
    // timer
    const tv = state === 'aim' ? Math.ceil(timer) : state === 'retreat' ? Math.ceil(timer) : '';
    ctx.fillStyle = 'rgba(10,15,30,.65)'; ctx.beginPath(); ctx.arc(W / 2, 48, 32, 0, Math.PI * 2); ctx.fill();
    text(String(tv), W / 2, 60, 32, state === 'retreat' ? '#ffd34d' : timer <= 5 ? '#ff5a4e' : '#fff');
    // wind
    ctx.fillStyle = 'rgba(10,15,30,.65)'; rr(ctx, W / 2 - 100, 90, 200, 30, 8); ctx.fill();
    ctx.fillStyle = 'rgba(255,255,255,.2)'; ctx.fillRect(W / 2 - 80, 102, 160, 6);
    ctx.fillStyle = '#8fe3ff';
    const ww = wind * 80; ctx.fillRect(ww < 0 ? W / 2 + ww : W / 2, 100, Math.abs(ww), 10);
    text('WIND', W / 2, 135, 12, '#cfe');
    // weapons
    if (state !== 'over') {
      const bw = 150, bx = W / 2 - (bw * 3 + 20) / 2;
      WEAPONS.forEach((w, i) => {
        const x = bx + i * (bw + 10), sel = i === weapon;
        ctx.fillStyle = sel ? 'rgba(255,211,77,.9)' : 'rgba(10,15,30,.65)'; rr(ctx, x, H - 46, bw, 34, 8); ctx.fill();
        let label = `${w.key}  ${w.name}`;
        if (w.id === 'airstrike' && active) label += ` ×${teamAmmo[active.team]}`;
        text(label, x + bw / 2, H - 23, 15, sel ? '#222' : '#fff');
      });
      let hint = '← → move  ·  ↑ ↓ aim  ·  Enter jump  ·  hold Space to fire';
      if (WEAPONS[weapon].id === 'airstrike') hint = 'Click the map to call the air strike';
      if (ai) hint = 'CPU is thinking…';
      if (state === 'retreat') hint = 'Retreat!';
      text(hint, W / 2, H - 60, 14, '#fff');
    }
    if (banner.t > 0) {
      ctx.globalAlpha = clamp(banner.t, 0, 1);
      text(banner.text, W / 2, 190, 30, '#fff');
      ctx.globalAlpha = 1;
    }
  }

  function drawOverlay(title, lines, btns) {
    ctx.fillStyle = 'rgba(5,10,25,.6)'; ctx.fillRect(0, 0, W, H);
    text(title, W / 2, H / 2 - 120, 72, '#ffd34d');
    lines.forEach((l, i) => text(l, W / 2, H / 2 - 60 + i * 30, 20, '#fff'));
    buttons = btns.map((b, i) => ({ ...b, x: W / 2 - 170, y: H / 2 + 20 + i * 70, w: 340, h: 54 }));
    for (const b of buttons) {
      const hov = mouse.x > b.x && mouse.x < b.x + b.w && mouse.y > b.y && mouse.y < b.y + b.h;
      ctx.fillStyle = hov ? '#ffd34d' : '#3d8bff'; rr(ctx, b.x, b.y, b.w, b.h, 12); ctx.fill();
      text(b.label, W / 2, b.y + 35, 22, hov ? '#222' : '#fff');
    }
  }

  function draw() {
    ctx.save();
    if (shake > 0.5) ctx.translate(rand(-shake, shake), rand(-shake, shake));
    ctx.drawImage(bgcv, 0, 0);
    drawClouds();
    ctx.drawImage(tcv, 0, 0);
    for (const u of units) if (u.alive) drawBot(u, u === active && state !== 'over' && state !== 'menu');
    for (const p of projs) drawProj(p);
    for (const p of parts) {
      ctx.fillStyle = p.color + clamp(p.life / p.max, 0, 1) + ')';
      ctx.beginPath(); ctx.arc(p.x, p.y, p.size, 0, Math.PI * 2); ctx.fill();
    }
    drawWater();
    drawAim();
    for (const p of pops) { ctx.globalAlpha = clamp(p.t, 0, 1); text(p.text, p.x, p.y, 20, p.color); }
    ctx.globalAlpha = 1;
    ctx.restore();

    if (state === 'menu') {
      drawOverlay('BOLT BARRAGE', ['Turn-based robot artillery. Last team standing wins.'], [
        { label: '1 — Play vs CPU', act: () => newGame(true) },
        { label: '2 — Two players (hot-seat)', act: () => newGame(false) },
      ]);
    } else if (state === 'over') {
      drawOverlay(winner < 0 ? 'DRAW!' : `${TEAMS[winner].name} win!`, [], [
        { label: 'R — Rematch', act: () => newGame(cpuMode) },
        { label: 'M — Main menu', act: () => { state = 'menu'; } },
      ]);
    } else {
      buttons = [];
      drawHUD();
    }
  }

  // ---------- input ----------
  addEventListener('keydown', e => {
    if (['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Space', 'Enter'].includes(e.code)) e.preventDefault();
    keys[e.code] = true;
    if (state === 'menu') {
      if (e.code === 'Digit1') newGame(true);
      if (e.code === 'Digit2') newGame(false);
      return;
    }
    if (state === 'over') {
      if (e.code === 'KeyR') newGame(cpuMode);
      if (e.code === 'KeyM') state = 'menu';
      return;
    }
    if (ai) return;
    if ((state === 'aim' || state === 'retreat') && e.code === 'Enter' && !charging) jump();
    if (state !== 'aim' || charging) return;
    const wi = ['Digit1', 'Digit2', 'Digit3'].indexOf(e.code);
    if (wi >= 0) weapon = wi;
    if (e.code === 'Space' && !e.repeat && WEAPONS[weapon].id !== 'airstrike') { charging = true; power = 0; }
  });
  addEventListener('keyup', e => {
    keys[e.code] = false;
    if (e.code === 'Space' && charging && state === 'aim' && !ai) fire();
  });
  function toWorld(e) {
    const r = cv.getBoundingClientRect();
    mouse.x = (e.clientX - r.left) * W / r.width; mouse.y = (e.clientY - r.top) * H / r.height;
  }
  cv.addEventListener('mousemove', toWorld);
  cv.addEventListener('mousedown', e => {
    toWorld(e);
    for (const b of buttons) {
      if (mouse.x > b.x && mouse.x < b.x + b.w && mouse.y > b.y && mouse.y < b.y + b.h) { b.act(); return; }
    }
    if (state === 'aim' && !ai && WEAPONS[weapon].id === 'airstrike') fire(mouse.x);
  });
  addEventListener('blur', () => { for (const k in keys) keys[k] = false; });

  // ---------- loop ----------
  genBackground(); genTerrain();
  let last = performance.now(), acc = 0;
  function frame(now) {
    acc += Math.min(0.1, (now - last) / 1000); last = now;
    while (acc >= STEP) { update(STEP); acc -= STEP; }
    draw();
    requestAnimationFrame(frame);
  }
  requestAnimationFrame(frame);

  // small hook for automated tests
  window.BoltBarrage = {
    get state() { return state; }, get units() { return units; }, get winner() { return winner; },
    newGame, setCpuBoth(v) { cpuMode = v; },
  };
})();
