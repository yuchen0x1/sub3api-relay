<template>
  <div class="stampede-backdrop" aria-hidden="true">
    <div class="stampede-photo stampede-photo-far" />
    <div class="stampede-photo stampede-photo-mid" />
    <div class="stampede-photo stampede-photo-near" />
    <div class="stampede-cinematic-wash" />
    <div class="stampede-energy-ribbon stampede-energy-ribbon-one" />
    <div class="stampede-energy-ribbon stampede-energy-ribbon-two" />
    <div class="stampede-speedlines" />
    <div class="stampede-dust-bank stampede-dust-bank-one" />
    <div class="stampede-dust-bank stampede-dust-bank-two" />
    <div class="stampede-particles">
      <span
        v-for="particle in particles"
        :key="particle.id"
        class="stampede-particle"
        :style="particleStyle(particle)"
      />
    </div>
    <div class="stampede-horizon" />
  </div>
</template>

<script setup lang="ts">
import type { CSSProperties } from 'vue'

interface Particle {
  id: number
  x: number
  y: number
  size: number
  duration: number
  delay: number
  drift: number
  hue: number
}

const particles: Particle[] = Array.from({ length: 78 }, (_, index) => ({
  id: index + 1,
  x: (index * 37) % 100,
  y: 22 + ((index * 17) % 68),
  size: 1 + (index % 4) * 0.8,
  duration: 5 + (index % 7) * 1.15,
  delay: -(index % 11) * 0.8,
  drift: 18 + (index % 6) * 12,
  hue: index % 3 === 0 ? 190 : index % 3 === 1 ? 258 : 276,
}))

function particleStyle(particle: Particle): CSSProperties {
  return {
    '--particle-x': `${particle.x}%`,
    '--particle-y': `${particle.y}%`,
    '--particle-size': `${particle.size}px`,
    '--particle-duration': `${particle.duration}s`,
    '--particle-delay': `${particle.delay}s`,
    '--particle-drift': `${particle.drift}px`,
    '--particle-hue': String(particle.hue),
  } as CSSProperties
}
</script>
