<script setup lang="ts">
import {House} from "@element-plus/icons-vue"
import {useRouter} from "vue-router"

const router = useRouter()

const goHome = () => {
	router.push("/")
}
</script>

<template>
	<div class="undefined-page">
		<div class="undefined-card">
			<span class="undefined-code">404</span>
			<h1>页面不存在</h1>
			<p>当前地址没有匹配到可用页面，你可以返回主页继续使用应用。</p>
			<el-button type="primary" :icon="House" @click="goHome">返回主页</el-button>
		</div>
	</div>
</template>

<style scoped>
.undefined-page {
	height: 100%;
	display: flex;
	align-items: center;
	justify-content: center;
}

.undefined-card {
	width: min(520px, 100%);
	padding: clamp(28px, 4vw, 40px);
	border-radius: var(--radius-2xl);
	background: var(--bg-card);
	border: 1px solid var(--border-strong);
	box-shadow: var(--shadow-soft);
	text-align: center;
}

.undefined-code {
	display: inline-flex;
	margin-bottom: var(--space-3);
	padding: 6px 12px;
	border-radius: 999px;
	background: var(--color-primary-soft);
	color: var(--color-primary);
	font-size: var(--font-size-xs);
	letter-spacing: 0.08em;
}

.undefined-card h1 {
	margin: 0 0 var(--space-3);
	font-size: clamp(28px, 4vw, 40px);
	line-height: var(--line-height-compact);
	color: var(--text-primary);
}

.undefined-card p {
	margin: 0 0 var(--space-6);
	color: var(--text-secondary);
	font-size: var(--font-size-md);
	line-height: var(--line-height-relaxed);
}
</style>