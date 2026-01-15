<template>
  <div class="asciinema-player-container">
    <div class="asciinema-player-wrapper" ref="playerRef"></div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, onBeforeUnmount } from 'vue'

interface Props {
  src: string
  cols?: number
  rows?: number
  autoplay?: boolean
  preload?: boolean
  startTime?: number
  speed?: number
  loop?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  cols: 80,
  rows: 24,
  autoplay: false,
  preload: true,
  startTime: 0,
  speed: 1,
  loop: false
})

const emit = defineEmits(['ready', 'play', 'pause', 'finish', 'progress'])

const playerRef = ref<HTMLDivElement>()
let player: any = null

// 动态加载 AsciinemaPlayer
const loadAsciinemaPlayer = async () => {
  return new Promise<void>((resolve, reject) => {
    const win = window as any
    // 检查是否已加载
    if (win.AsciinemaPlayer || win.AsciiinemaPlayer) {
      resolve()
      return
    }

    // 加载 CSS
    const css = document.createElement('link')
    css.rel = 'stylesheet'
    css.href = 'https://cdn.jsdelivr.net/npm/asciinema-player@3.6.3/dist/bundle/asciinema-player.css'
    document.head.appendChild(css)

    // 加载 JS
    const script = document.createElement('script')
    script.src = 'https://cdn.jsdelivr.net/npm/asciinema-player@3.6.3/dist/bundle/asciinema-player.min.js'
    script.onload = () => {
      console.log('✅ AsciinemaPlayer 库已加载')
      console.log('全局变量:', {
        AsciinemaPlayer: win.AsciinemaPlayer,
        AsciiinemaPlayer: win.AsciiinemaPlayer
      })
      resolve()
    }
    script.onerror = () => reject(new Error('Failed to load AsciinemaPlayer'))
    document.head.appendChild(script)
  })
}

// 创建播放器
const createPlayer = async () => {
  if (!playerRef.value || !props.src) return

  try {
    await loadAsciinemaPlayer()

    // 清除旧播放器
    if (player) {
      playerRef.value.innerHTML = ''
    }

    const win = window as any
    // 尝试两种可能的全局变量名
    const AsciinemaPlayerLibrary = win.AsciinemaPlayer || win.AsciiinemaPlayer

    console.log('📼 AsciinemaPlayer 库:', AsciinemaPlayerLibrary)
    console.log('📼 播放器容器:', playerRef.value)
    console.log('📼 录制文件 URL:', props.src)

    if (!AsciinemaPlayerLibrary) {
      throw new Error('AsciinemaPlayer library not loaded')
    }

    // 使用 create 函数创建播放器（asciinema-player v3+）
    player = AsciinemaPlayerLibrary.create(props.src, playerRef.value, {
      // 不设置 cols 和 rows，让播放器从录制文件中自动读取
      autoplay: props.autoplay,
      preload: props.preload ? 'auto' : 'none',
      startTime: props.startTime,
      speed: props.speed,
      loop: props.loop,
      theme: 'tango',
      poster: 'npt:0:01',
      // 确保控制栏显示
      controls: true,
    })

    console.log('✅ 播放器创建成功:', player)

    // 监听事件
    if (player.addEventListener) {
      player.addEventListener('ready', () => emit('ready'))
      player.addEventListener('play', () => emit('play'))
      player.addEventListener('pause', () => emit('pause'))
      player.addEventListener('ended', () => emit('finish'))
      player.addEventListener('progress', (e: any) => emit('progress', e))
    }

    emit('ready')
  } catch (error) {
    console.error('❌ Failed to create AsciinemaPlayer:', error)
    console.error('错误详情:', error)
  }
}

// 监听 src 变化
watch(() => props.src, () => {
  createPlayer()
})

onMounted(() => {
  createPlayer()
})

onBeforeUnmount(() => {
  if (player && playerRef.value) {
    playerRef.value.innerHTML = ''
    player = null
  }
})

// 暴露方法
defineExpose({
  play: () => player?.play(),
  pause: () => player?.pause(),
  seek: (time: number) => player?.seek(time),
  getDuration: () => player?.duration,
  getCurrentTime: () => player?.currentTime,
})
</script>

<style scoped>
.asciinema-player-container {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #000;
  min-height: 500px;
}

.asciinema-player-wrapper {
  width: 100%;
  height: 100%;
  overflow: auto;
}

/* 深度样式覆盖 - 修改 AsciinemaPlayer 的颜色 */
.asciinema-player-wrapper :deep(.asciinema-player) {
  background-color: #000 !important;
}

.asciinema-player-wrapper :deep(.asciinema-player .ap-terminal) {
  background-color: #000 !important;
}

.asciinema-player-wrapper :deep(.asciinema-player .ap-control-bar) {
  background: rgba(0, 0, 0, 0.9) !important;
  opacity: 1 !important;
  height: auto !important;
  min-height: 48px !important;
}

.asciinema-player-wrapper :deep(.asciinema-player .ap-progress-container) {
  background-color: rgba(212, 175, 55, 0.2) !important;
  height: 6px !important;
}

.asciinema-player-wrapper :deep(.asciinema-player .ap-progress-bar) {
  background-color: #d4af37 !important;
}

.asciinema-player-wrapper :deep(.asciinema-player .ap-controls) {
  color: #d4af37 !important;
  display: flex !important;
  opacity: 1 !important;
}

.asciinema-player-wrapper :deep(.asciinema-player .ap-control-bar) {
  display: block !important;
}

.asciinema-player-wrapper :deep(.asciinema-player .ap-icon-button) {
  display: inline-flex !important;
  color: #d4af37 !important;
}

.asciinema-player-wrapper :deep(.asciinema-player .ap-icon-button:hover) {
  color: #bfa13f !important;
}
</style>
