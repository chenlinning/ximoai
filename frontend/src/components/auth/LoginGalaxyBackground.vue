<template>
  <div ref="containerRef" class="login-galaxy-background" aria-hidden="true"></div>
</template>

<script setup lang="ts">
import { Color, Mesh, Program, Renderer, Triangle } from 'ogl'
import { onBeforeUnmount, onMounted, ref } from 'vue'

const containerRef = ref<HTMLDivElement | null>(null)

const focal = [0.5, 0.5]
const rotation = [1.0, 0.0]
const starSpeed = 0.5
const disableAnimation = false
const speed = 1.0
const mouseInteraction = true
const mouseRepulsion = true
const repulsionStrength = 2
const rotationSpeed = 0.1
const autoCenterRepulsion = 0
const transparent = true

let renderer: Renderer | null = null
let program: Program | null = null
let mesh: Mesh | null = null
let animationFrame = 0
let themeObserver: MutationObserver | null = null

const targetMousePos = { x: 0.5, y: 0.5 }
const smoothMousePos = { x: 0.5, y: 0.5 }
let targetMouseActive = 0.0
let smoothMouseActive = 0.0

type GalaxyTheme = {
  density: number
  hueShift: number
  glowIntensity: number
  saturation: number
  twinkleIntensity: number
}

const lightTheme: GalaxyTheme = {
  density: 0.45,
  hueShift: 128,
  glowIntensity: 0.08,
  saturation: 0.35,
  twinkleIntensity: 0.1
}

const darkTheme: GalaxyTheme = {
  density: 1.05,
  hueShift: 128,
  glowIntensity: 0.38,
  saturation: 0.48,
  twinkleIntensity: 0.32
}

function getGalaxyTheme() {
  return document.documentElement.classList.contains('dark') ? darkTheme : lightTheme
}

function getThemeBackground() {
  const styles = getComputedStyle(document.documentElement)
  const token = document.documentElement.classList.contains('dark') ? '--color-dark-950' : '--color-accent-50'
  const rgb = styles.getPropertyValue(token).trim()
  return rgb ? `rgb(${rgb})` : document.documentElement.classList.contains('dark') ? '#171717' : '#faf9f6'
}

function applyGalaxyTheme() {
  const container = containerRef.value
  const theme = getGalaxyTheme()
  if (container) {
    container.style.background = getThemeBackground()
  }

  if (!program) return
  program.uniforms.uDensity.value = theme.density
  program.uniforms.uHueShift.value = theme.hueShift
  program.uniforms.uGlowIntensity.value = theme.glowIntensity
  program.uniforms.uSaturation.value = theme.saturation
  program.uniforms.uTwinkleIntensity.value = theme.twinkleIntensity
}

const vertexShader = `
attribute vec2 uv;
attribute vec2 position;

varying vec2 vUv;

void main() {
  vUv = uv;
  gl_Position = vec4(position, 0, 1);
}
`

const fragmentShader = `
precision highp float;

uniform float uTime;
uniform vec3 uResolution;
uniform vec2 uFocal;
uniform vec2 uRotation;
uniform float uStarSpeed;
uniform float uDensity;
uniform float uHueShift;
uniform float uSpeed;
uniform vec2 uMouse;
uniform float uGlowIntensity;
uniform float uSaturation;
uniform bool uMouseRepulsion;
uniform float uTwinkleIntensity;
uniform float uRotationSpeed;
uniform float uRepulsionStrength;
uniform float uMouseActiveFactor;
uniform float uAutoCenterRepulsion;
uniform bool uTransparent;

varying vec2 vUv;

#define NUM_LAYER 4.0
#define STAR_COLOR_CUTOFF 0.2
#define MAT45 mat2(0.7071, -0.7071, 0.7071, 0.7071)
#define PERIOD 3.0

float Hash21(vec2 p) {
  p = fract(p * vec2(123.34, 456.21));
  p += dot(p, p + 45.32);
  return fract(p.x * p.y);
}

float tri(float x) {
  return abs(fract(x) * 2.0 - 1.0);
}

float tris(float x) {
  float t = fract(x);
  return 1.0 - smoothstep(0.0, 1.0, abs(2.0 * t - 1.0));
}

float trisn(float x) {
  float t = fract(x);
  return 2.0 * (1.0 - smoothstep(0.0, 1.0, abs(2.0 * t - 1.0))) - 1.0;
}

vec3 hsv2rgb(vec3 c) {
  vec4 K = vec4(1.0, 2.0 / 3.0, 1.0 / 3.0, 3.0);
  vec3 p = abs(fract(c.xxx + K.xyz) * 6.0 - K.www);
  return c.z * mix(K.xxx, clamp(p - K.xxx, 0.0, 1.0), c.y);
}

float Star(vec2 uv, float flare) {
  float d = length(uv);
  float m = (0.05 * uGlowIntensity) / d;
  float rays = smoothstep(0.0, 1.0, 1.0 - abs(uv.x * uv.y * 1000.0));
  m += rays * flare * uGlowIntensity;
  uv *= MAT45;
  rays = smoothstep(0.0, 1.0, 1.0 - abs(uv.x * uv.y * 1000.0));
  m += rays * 0.3 * flare * uGlowIntensity;
  m *= smoothstep(1.0, 0.2, d);
  return m;
}

vec3 StarLayer(vec2 uv) {
  vec3 col = vec3(0.0);

  vec2 gv = fract(uv) - 0.5;
  vec2 id = floor(uv);

  for (int y = -1; y <= 1; y++) {
    for (int x = -1; x <= 1; x++) {
      vec2 offset = vec2(float(x), float(y));
      vec2 si = id + vec2(float(x), float(y));
      float seed = Hash21(si);
      float size = fract(seed * 345.32);
      float glossLocal = tri(uStarSpeed / (PERIOD * seed + 1.0));
      float flareSize = smoothstep(0.9, 1.0, size) * glossLocal;

      float red = smoothstep(STAR_COLOR_CUTOFF, 1.0, Hash21(si + 1.0)) + STAR_COLOR_CUTOFF;
      float blu = smoothstep(STAR_COLOR_CUTOFF, 1.0, Hash21(si + 3.0)) + STAR_COLOR_CUTOFF;
      float grn = min(red, blu) * seed;
      vec3 base = vec3(red, grn, blu);

      float hue = atan(base.g - base.r, base.b - base.r) / (2.0 * 3.14159) + 0.5;
      hue = fract(hue + uHueShift / 360.0);
      float sat = length(base - vec3(dot(base, vec3(0.299, 0.587, 0.114)))) * uSaturation;
      float val = max(max(base.r, base.g), base.b);
      base = hsv2rgb(vec3(hue, sat, val));

      vec2 pad = vec2(tris(seed * 34.0 + uTime * uSpeed / 10.0), tris(seed * 38.0 + uTime * uSpeed / 30.0)) - 0.5;

      float star = Star(gv - offset - pad, flareSize);
      vec3 color = base;

      float twinkle = trisn(uTime * uSpeed + seed * 6.2831) * 0.5 + 1.0;
      twinkle = mix(1.0, twinkle, uTwinkleIntensity);
      star *= twinkle;

      col += star * size * color;
    }
  }

  return col;
}

void main() {
  vec2 focalPx = uFocal * uResolution.xy;
  vec2 uv = (vUv * uResolution.xy - focalPx) / uResolution.y;

  vec2 mouseNorm = uMouse - vec2(0.5);

  if (uAutoCenterRepulsion > 0.0) {
    vec2 centerUV = vec2(0.0, 0.0);
    float centerDist = length(uv - centerUV);
    vec2 repulsion = normalize(uv - centerUV) * (uAutoCenterRepulsion / (centerDist + 0.1));
    uv += repulsion * 0.05;
  } else if (uMouseRepulsion) {
    vec2 mousePosUV = (uMouse * uResolution.xy - focalPx) / uResolution.y;
    float mouseDist = length(uv - mousePosUV);
    vec2 repulsion = normalize(uv - mousePosUV) * (uRepulsionStrength / (mouseDist + 0.1));
    uv += repulsion * 0.05 * uMouseActiveFactor;
  } else {
    vec2 mouseOffset = mouseNorm * 0.1 * uMouseActiveFactor;
    uv += mouseOffset;
  }

  float autoRotAngle = uTime * uRotationSpeed;
  mat2 autoRot = mat2(cos(autoRotAngle), -sin(autoRotAngle), sin(autoRotAngle), cos(autoRotAngle));
  uv = autoRot * uv;

  uv = mat2(uRotation.x, -uRotation.y, uRotation.y, uRotation.x) * uv;

  vec3 col = vec3(0.0);

  for (float i = 0.0; i < 1.0; i += 1.0 / NUM_LAYER) {
    float depth = fract(i + uStarSpeed * uSpeed);
    float scale = mix(20.0 * uDensity, 0.5 * uDensity, depth);
    float fade = depth * smoothstep(1.0, 0.9, depth);
    col += StarLayer(uv * scale + i * 453.32) * fade;
  }

  if (uTransparent) {
    float alpha = length(col);
    alpha = smoothstep(0.0, 0.3, alpha);
    alpha = min(alpha, 1.0);
    gl_FragColor = vec4(col, alpha);
  } else {
    gl_FragColor = vec4(col, 1.0);
  }
}
`

function resize() {
  const container = containerRef.value
  if (!container || !renderer) return

  renderer.setSize(container.offsetWidth, container.offsetHeight)
  if (program) {
    const canvas = renderer.gl.canvas
    program.uniforms.uResolution.value = new Color(canvas.width, canvas.height, canvas.width / canvas.height)
  }
}

function update(time: number) {
  animationFrame = requestAnimationFrame(update)
  if (!renderer || !program || !mesh) return

  if (!disableAnimation) {
    program.uniforms.uTime.value = time * 0.001
    program.uniforms.uStarSpeed.value = (time * 0.001 * starSpeed) / 10.0
  }

  const lerpFactor = 0.05
  smoothMousePos.x += (targetMousePos.x - smoothMousePos.x) * lerpFactor
  smoothMousePos.y += (targetMousePos.y - smoothMousePos.y) * lerpFactor
  smoothMouseActive += (targetMouseActive - smoothMouseActive) * lerpFactor

  program.uniforms.uMouse.value[0] = smoothMousePos.x
  program.uniforms.uMouse.value[1] = smoothMousePos.y
  program.uniforms.uMouseActiveFactor.value = smoothMouseActive

  renderer.render({ scene: mesh })
}

function handleMouseMove(event: MouseEvent) {
  const container = containerRef.value
  if (!container) return

  const rect = container.getBoundingClientRect()
  targetMousePos.x = (event.clientX - rect.left) / rect.width
  targetMousePos.y = 1.0 - (event.clientY - rect.top) / rect.height
  targetMouseActive = 1.0
}

function handleMouseLeave() {
  targetMouseActive = 0.0
}

function mountGalaxy() {
  const container = containerRef.value
  if (!container) return

  renderer = new Renderer({
    alpha: transparent,
    premultipliedAlpha: false
  })

  const gl = renderer.gl
  gl.canvas.className = 'login-galaxy-canvas'

  if (transparent) {
    gl.enable(gl.BLEND)
    gl.blendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
    gl.clearColor(0, 0, 0, 0)
  } else {
    gl.clearColor(0, 0, 0, 1)
  }

  resize()

  const geometry = new Triangle(gl)
  const theme = getGalaxyTheme()
  program = new Program(gl, {
    vertex: vertexShader,
    fragment: fragmentShader,
    uniforms: {
      uTime: { value: 0 },
      uResolution: { value: new Color(gl.canvas.width, gl.canvas.height, gl.canvas.width / gl.canvas.height) },
      uFocal: { value: new Float32Array(focal) },
      uRotation: { value: new Float32Array(rotation) },
      uStarSpeed: { value: starSpeed },
      uDensity: { value: theme.density },
      uHueShift: { value: theme.hueShift },
      uSpeed: { value: speed },
      uMouse: { value: new Float32Array([smoothMousePos.x, smoothMousePos.y]) },
      uGlowIntensity: { value: theme.glowIntensity },
      uSaturation: { value: theme.saturation },
      uMouseRepulsion: { value: mouseRepulsion },
      uTwinkleIntensity: { value: theme.twinkleIntensity },
      uRotationSpeed: { value: rotationSpeed },
      uRepulsionStrength: { value: repulsionStrength },
      uMouseActiveFactor: { value: 0.0 },
      uAutoCenterRepulsion: { value: autoCenterRepulsion },
      uTransparent: { value: transparent }
    }
  })

  mesh = new Mesh(gl, { geometry, program })
  container.appendChild(gl.canvas)
  window.addEventListener('resize', resize, false)
  themeObserver = new MutationObserver(applyGalaxyTheme)
  themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
  applyGalaxyTheme()

  if (mouseInteraction) {
    container.addEventListener('mousemove', handleMouseMove)
    container.addEventListener('mouseleave', handleMouseLeave)
  }

  animationFrame = requestAnimationFrame(update)
}

onMounted(mountGalaxy)

onBeforeUnmount(() => {
  const container = containerRef.value
  cancelAnimationFrame(animationFrame)
  window.removeEventListener('resize', resize)
  themeObserver?.disconnect()
  themeObserver = null

  if (mouseInteraction && container) {
    container.removeEventListener('mousemove', handleMouseMove)
    container.removeEventListener('mouseleave', handleMouseLeave)
  }

  const canvas = renderer?.gl.canvas
  if (canvas?.parentNode === container) {
    container?.removeChild(canvas)
  }
  renderer?.gl.getExtension('WEBGL_lose_context')?.loseContext()

  renderer = null
  program = null
  mesh = null
})
</script>

<style scoped>
.login-galaxy-background {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  overflow: hidden;
  background: var(--login-galaxy-background, rgb(var(--color-accent-50)));
}

:deep(.login-galaxy-canvas) {
  display: block;
  width: 100%;
  height: 100%;
}
</style>
