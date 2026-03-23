<script setup lang="ts">
import {ArrowRight, ChatLineSquare, Setting} from '@element-plus/icons-vue'
import {useRouter} from 'vue-router'
import {showSetting} from '@/service/ApplicationService'

const router = useRouter()

const openAssistant = () => {
  router.push('/chat')
}

const openSetting = () => {
  showSetting.value = true
}
</script>

<template>
  <div class="home-page">
    <section class="hero-panel">
      <div class="hero-copy">
        <span class="hero-badge">桌面工作台</span>
        <h1>一个入口，先把智能助理接进来。</h1>
        <p>
          当前版本聚焦一个核心能力：通过火山引擎大模型为桌面应用提供智能对话入口。
          默认模型来自设置，新建会话时自动采用该配置，并允许在会话中按需切换。
        </p>
        <div class="hero-actions">
          <el-button type="primary" size="large" @click="openAssistant">
            打开智能助理
            <el-icon class="button-icon"><ArrowRight /></el-icon>
          </el-button>
          <el-button size="large" @click="openSetting">
            <el-icon><Setting /></el-icon>
            配置模型
          </el-button>
        </div>
      </div>
      <div class="hero-metric">
        <span>已接入能力</span>
        <strong>1</strong>
        <p>智能助理</p>
      </div>
    </section>

    <section class="feature-grid">
      <article class="feature-card" @click="openAssistant">
        <div class="feature-card__icon">
          <el-icon><ChatLineSquare /></el-icon>
        </div>
        <div class="feature-card__content">
          <h2>智能助理</h2>
          <p>支持会话式问答、流式回复和会话级模型切换，适合作为当前应用的第一能力入口。</p>
        </div>
        <span class="feature-card__action">立即进入</span>
      </article>
    </section>
  </div>
</template>

<style scoped>
.home-page {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.hero-panel {
  display: grid;
  grid-template-columns: minmax(0, 1.15fr) minmax(240px, 320px);
  gap: var(--space-6);
  padding: clamp(24px, 3vw, 36px);
  border-radius: var(--radius-2xl);
  background: var(--hero-gradient);
  color: var(--text-on-primary-strong);
  box-shadow: var(--shadow-glow);
  min-height: 280px;
}

.hero-copy h1 {
  margin: var(--space-4) 0 var(--space-3);
  font-size: clamp(36px, 4vw, 52px);
  line-height: 1.08;
}

.hero-copy p {
  max-width: 860px;
  margin: 0;
  font-size: var(--font-size-md);
  line-height: var(--line-height-relaxed);
  color: var(--text-on-primary);
}

.hero-badge {
  display: inline-flex;
  padding: 6px 12px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.12);
  font-size: var(--font-size-xs);
  letter-spacing: 0.08em;
}

.hero-actions {
  display: flex;
  gap: var(--space-3);
  margin-top: var(--space-6);
  flex-wrap: wrap;
}

.button-icon {
  margin-left: 6px;
}

.hero-metric {
  border-radius: var(--radius-xl);
  padding: var(--space-5);
  background: rgba(255, 255, 255, 0.12);
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  backdrop-filter: blur(12px);
}

.hero-metric span,
.hero-metric p {
  color: var(--text-on-primary);
}

.hero-metric strong {
  font-size: clamp(68px, 8vw, 92px);
  line-height: 1;
}

.feature-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: var(--space-5);
  flex: 1;
}

.feature-card {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: var(--space-5);
  padding: clamp(22px, 2vw, 30px);
  border-radius: var(--radius-2xl);
  background: var(--bg-card);
  border: 1px solid var(--border-strong);
  box-shadow: var(--shadow-soft);
  cursor: pointer;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.feature-card:hover {
  transform: translateY(-3px);
  box-shadow: var(--shadow-hover);
}

.feature-card__icon {
  width: 60px;
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-lg);
  background: var(--color-primary-soft);
  color: var(--color-primary);
  font-size: 30px;
}

.feature-card__content h2 {
  margin: 0 0 var(--space-2);
  font-size: var(--font-size-xl);
  color: var(--text-primary);
}

.feature-card__content p {
  margin: 0;
  color: var(--text-tertiary);
  line-height: var(--line-height-relaxed);
}

.feature-card__action {
  color: var(--color-primary);
  font-weight: 600;
  white-space: nowrap;
}

@media (max-width: 960px) {
  .hero-panel {
    grid-template-columns: 1fr;
  }

  .feature-card {
    grid-template-columns: 1fr;
    align-items: start;
  }
}
</style>