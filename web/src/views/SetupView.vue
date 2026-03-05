<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  NCard,
  NButton,
  NForm,
  NFormItem,
  NInput,
  NSwitch,
  NSteps,
  NStep,
  NSpace,
  NAlert,
  useMessage,
} from 'naive-ui'
import { api, type AppConfig } from '../api'

const message = useMessage()
const currentStep = ref(1)
const loading = ref(false)
const testResult = ref<{ success: boolean; message: string } | null>(null)

const config = ref<AppConfig>({
  server: { port: 8096 },
  webdav: { url: '', user: '', password: '', vendor: 'other' },
  backup: { library_dir: '/data/library', backups_dir: '/data/backups', remote_dir: '/immich-backup' },
  encrypt: { enabled: false, password: '', salt: '' },
  cron: { enabled: false, expression: '0 2 * * *' },
})

async function loadConfig() {
  try {
    config.value = await api.getConfig()
  } catch {
    // 首次使用，保持默认值
  }
}

async function testConnection() {
  loading.value = true
  testResult.value = null
  try {
    const result = await api.testWebDAV({
      url: config.value.webdav.url,
      user: config.value.webdav.user,
      password: config.value.webdav.password,
    })
    testResult.value = result
    if (result.success) {
      message.success('连接成功！')
    } else {
      message.error('连接失败: ' + result.message)
    }
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e)
    message.error('测试失败: ' + msg)
  } finally {
    loading.value = false
  }
}

async function saveConfig() {
  loading.value = true
  try {
    await api.saveConfig(config.value)
    message.success('配置已保存')
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e)
    message.error('保存失败: ' + msg)
  } finally {
    loading.value = false
  }
}

function nextStep() {
  if (currentStep.value < 4) currentStep.value++
}

function prevStep() {
  if (currentStep.value > 1) currentStep.value--
}

onMounted(loadConfig)
</script>

<template>
  <div>
    <h2 class="text-2xl font-bold text-gray-800 mb-6">任务配置</h2>

    <NSteps :current="currentStep" class="mb-8">
      <NStep title="WebDAV 连接" />
      <NStep title="加密设置" />
      <NStep title="备份目录" />
      <NStep title="定时任务" />
    </NSteps>

    <!-- Step 1: WebDAV -->
    <NCard v-if="currentStep === 1" title="WebDAV 连接配置">
      <NForm label-placement="left" label-width="120">
        <NFormItem label="WebDAV URL">
          <NInput v-model:value="config.webdav.url" placeholder="https://dav.115.com/..." />
        </NFormItem>
        <NFormItem label="用户名">
          <NInput v-model:value="config.webdav.user" placeholder="WebDAV 用户名" />
        </NFormItem>
        <NFormItem label="密码">
          <NInput
            v-model:value="config.webdav.password"
            type="password"
            show-password-on="click"
            placeholder="WebDAV 密码"
          />
        </NFormItem>
      </NForm>

      <NAlert v-if="testResult" :type="testResult.success ? 'success' : 'error'" class="mt-4">
        {{ testResult.message }}
      </NAlert>

      <NSpace class="mt-4">
        <NButton :loading="loading" @click="testConnection">🔗 测试连接</NButton>
        <NButton type="primary" @click="nextStep">下一步 →</NButton>
      </NSpace>
    </NCard>

    <!-- Step 2: 加密 -->
    <NCard v-if="currentStep === 2" title="加密设置 (Rclone Crypt)">
      <NAlert type="info" class="mb-4">
        启用加密后，上传到 115 网盘的文件和目录名将被加密。恢复时系统会自动解密。
      </NAlert>
      <NForm label-placement="left" label-width="120">
        <NFormItem label="启用加密">
          <NSwitch v-model:value="config.encrypt.enabled" />
        </NFormItem>
        <template v-if="config.encrypt.enabled">
          <NFormItem label="加密密码">
            <NInput
              v-model:value="config.encrypt.password"
              type="password"
              show-password-on="click"
              placeholder="用于加密文件内容的密码"
            />
          </NFormItem>
          <NFormItem label="加密盐值">
            <NInput
              v-model:value="config.encrypt.salt"
              type="password"
              show-password-on="click"
              placeholder="可选，增强安全性"
            />
          </NFormItem>
        </template>
      </NForm>
      <NSpace class="mt-4">
        <NButton @click="prevStep">← 上一步</NButton>
        <NButton type="primary" @click="nextStep">下一步 →</NButton>
      </NSpace>
    </NCard>

    <!-- Step 3: 备份目录 -->
    <NCard v-if="currentStep === 3" title="备份目录配置">
      <NAlert type="info" class="mb-4">
        Docker 用户请使用容器内的挂载路径（如 /data/library）。宿主机用户请使用绝对路径。
      </NAlert>
      <NForm label-placement="left" label-width="160">
        <NFormItem label="Immich Library 目录">
          <NInput v-model:value="config.backup.library_dir" placeholder="/data/library" />
        </NFormItem>
        <NFormItem label="Immich Backups 目录">
          <NInput v-model:value="config.backup.backups_dir" placeholder="/data/backups" />
        </NFormItem>
        <NFormItem label="远端目标目录">
          <NInput v-model:value="config.backup.remote_dir" placeholder="/immich-backup" />
        </NFormItem>
      </NForm>
      <NSpace class="mt-4">
        <NButton @click="prevStep">← 上一步</NButton>
        <NButton type="primary" @click="nextStep">下一步 →</NButton>
      </NSpace>
    </NCard>

    <!-- Step 4: 定时任务 -->
    <NCard v-if="currentStep === 4" title="定时任务">
      <NForm label-placement="left" label-width="120">
        <NFormItem label="启用定时">
          <NSwitch v-model:value="config.cron.enabled" />
        </NFormItem>
        <NFormItem v-if="config.cron.enabled" label="Cron 表达式">
          <NInput v-model:value="config.cron.expression" placeholder="0 2 * * *" />
        </NFormItem>
      </NForm>

      <NAlert v-if="config.cron.enabled" type="info" class="mt-2 mb-4">
        常用表达式：<br />
        <code>0 2 * * *</code> — 每天凌晨 2 点<br />
        <code>0 */6 * * *</code> — 每 6 小时<br />
        <code>0 3 * * 0</code> — 每周日凌晨 3 点
      </NAlert>

      <NSpace class="mt-4">
        <NButton @click="prevStep">← 上一步</NButton>
        <NButton type="primary" :loading="loading" @click="saveConfig">
          💾 保存全部配置
        </NButton>
      </NSpace>
    </NCard>
  </div>
</template>
