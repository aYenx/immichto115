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
  NAlert,
  useMessage,
  NIcon
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
  <div class="max-w-4xl mx-auto">
    <div class="flex items-center justify-between mb-8">
      <h2 class="text-3xl font-extrabold tracking-tight text-slate-900">任务配置</h2>
    </div>

    <NCard class="mb-8 shadow-sm hover:shadow-md transition-shadow">
      <NSteps :current="currentStep" status="process">
        <NStep title="WebDAV 连接" description="配置115网盘" />
        <NStep title="加密设置" description="保护数据安全" />
        <NStep title="备份目录" description="设置源与目标" />
        <NStep title="定时任务" description="自动化执行" />
      </NSteps>
    </NCard>

    <div class="relative overflow-hidden min-h-[400px]">
      <transition name="slide-fade" mode="out-in">
        <!-- Step 1: WebDAV -->
        <NCard v-if="currentStep === 1" title="WebDAV 连接配置" class="shadow-sm" :bordered="false" key="step1">
          <NForm label-placement="top" size="large" class="mt-4">
            <NFormItem label="WebDAV URL">
              <NInput v-model:value="config.webdav.url" placeholder="https://dav.115.com/..." class="font-mono text-sm" />
            </NFormItem>
            <NFormItem label="用户名">
              <NInput v-model:value="config.webdav.user" placeholder="输入 WebDAV 用户名" />
            </NFormItem>
            <NFormItem label="密码">
              <NInput
                v-model:value="config.webdav.password"
                type="password"
                show-password-on="click"
                placeholder="输入 WebDAV 密码"
              />
            </NFormItem>
          </NForm>

          <transition name="fade">
            <NAlert v-if="testResult" :type="testResult.success ? 'success' : 'error'" class="mt-4">
              {{ testResult.message }}
            </NAlert>
          </transition>

          <div class="flex justify-between items-center mt-8 pt-6 border-t border-slate-100">
            <NButton :loading="loading" @click="testConnection" secondary type="info">
              <template #icon>
                <NIcon><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg></NIcon>
              </template>
              测试连接
            </NButton>
            <NButton type="primary" size="large" @click="nextStep" class="px-8 shadow-sm shadow-blue-500/30">
              下一步
              <template #icon>
                <NIcon><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/></svg></NIcon>
              </template>
            </NButton>
          </div>
        </NCard>

        <!-- Step 2: 加密 -->
        <NCard v-else-if="currentStep === 2" title="加密设置 (Rclone Crypt)" class="shadow-sm" :bordered="false" key="step2">
          <NAlert type="info" class="mb-6 bg-blue-50 border border-blue-100">
            <template #icon>
              <NIcon><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg></NIcon>
            </template>
            启用加密后，上传到 115 网盘的文件和目录名将被加密。恢复时系统会自动解密。
          </NAlert>
          <NForm label-placement="left" label-width="120" size="large">
            <NFormItem label="启用加密">
              <NSwitch v-model:value="config.encrypt.enabled" />
            </NFormItem>
            <transition name="fade">
              <div v-if="config.encrypt.enabled" class="bg-slate-50 p-6 rounded-lg border border-slate-100 mt-4">
                <NFormItem label="加密密码" class="mb-4">
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
              </div>
            </transition>
          </NForm>
          <div class="flex justify-between items-center mt-8 pt-6 border-t border-slate-100">
            <NButton @click="prevStep" secondary>
              <template #icon>
                <NIcon><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="19" y1="12" x2="5" y2="12"/><polyline points="12 19 5 12 12 5"/></svg></NIcon>
              </template>
              上一步
            </NButton>
            <NButton type="primary" size="large" @click="nextStep" class="px-8 shadow-sm shadow-blue-500/30">
              下一步
              <template #icon>
                <NIcon><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/></svg></NIcon>
              </template>
            </NButton>
          </div>
        </NCard>

        <!-- Step 3: 备份目录 -->
        <NCard v-else-if="currentStep === 3" title="备份目录配置" class="shadow-sm" :bordered="false" key="step3">
          <NAlert type="info" class="mb-6 bg-blue-50 border border-blue-100">
            <template #icon>
              <NIcon><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg></NIcon>
            </template>
            Docker 用户请使用容器内的挂载路径（如 /data/library）。宿主机用户请使用绝对路径。
          </NAlert>
          <NForm label-placement="top" size="large">
            <NFormItem label="Immich Library 目录 (源)">
              <NInput v-model:value="config.backup.library_dir" placeholder="/data/library" class="font-mono text-sm" />
            </NFormItem>
            <NFormItem label="Immich Backups 目录 (源)">
              <NInput v-model:value="config.backup.backups_dir" placeholder="/data/backups" class="font-mono text-sm" />
            </NFormItem>
            <div class="h-px bg-slate-200 my-4"></div>
            <NFormItem label="远端目标目录 (115网盘)">
              <NInput v-model:value="config.backup.remote_dir" placeholder="/immich-backup" class="font-mono text-sm" />
            </NFormItem>
          </NForm>
          <div class="flex justify-between items-center mt-8 pt-6 border-t border-slate-100">
            <NButton @click="prevStep" secondary>
              <template #icon>
                <NIcon><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="19" y1="12" x2="5" y2="12"/><polyline points="12 19 5 12 12 5"/></svg></NIcon>
              </template>
              上一步
            </NButton>
            <NButton type="primary" size="large" @click="nextStep" class="px-8 shadow-sm shadow-blue-500/30">
              下一步
              <template #icon>
                <NIcon><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/></svg></NIcon>
              </template>
            </NButton>
          </div>
        </NCard>

        <!-- Step 4: 定时任务 -->
        <NCard v-else-if="currentStep === 4" title="定时任务配置" class="shadow-sm" :bordered="false" key="step4">
          <NForm label-placement="left" label-width="120" size="large">
            <NFormItem label="启用定时">
              <NSwitch v-model:value="config.cron.enabled" />
            </NFormItem>
            <transition name="fade">
              <div v-if="config.cron.enabled" class="bg-slate-50 p-6 rounded-lg border border-slate-100 mt-4">
                <NFormItem label="Cron 表达式" class="mb-0">
                  <NInput v-model:value="config.cron.expression" placeholder="0 2 * * *" class="font-mono text-sm" />
                </NFormItem>
              </div>
            </transition>
          </NForm>

          <transition name="fade">
            <NAlert v-if="config.cron.enabled" type="info" class="mt-6 bg-blue-50 border border-blue-100">
              <template #icon>
                <NIcon><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg></NIcon>
              </template>
              <span class="font-bold block mb-2">常用表达式：</span>
              <div class="space-y-1 text-sm">
                <div class="flex"><code class="bg-white px-2 py-0.5 rounded text-blue-600 font-mono text-xs mr-2 border border-blue-100">0 2 * * *</code> <span>每天凌晨 2 点</span></div>
                <div class="flex"><code class="bg-white px-2 py-0.5 rounded text-blue-600 font-mono text-xs mr-2 border border-blue-100">0 */6 * * *</code> <span>每 6 小时</span></div>
                <div class="flex"><code class="bg-white px-2 py-0.5 rounded text-blue-600 font-mono text-xs mr-2 border border-blue-100">0 3 * * 0</code> <span>每周日凌晨 3 点</span></div>
              </div>
            </NAlert>
          </transition>

          <div class="flex justify-between items-center mt-8 pt-6 border-t border-slate-100">
            <NButton @click="prevStep" secondary>
              <template #icon>
                <NIcon><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="19" y1="12" x2="5" y2="12"/><polyline points="12 19 5 12 12 5"/></svg></NIcon>
              </template>
              上一步
            </NButton>
            <NButton type="primary" size="large" :loading="loading" @click="saveConfig" class="px-8 shadow-md shadow-emerald-500/40 bg-emerald-500 hover:bg-emerald-600 border-none">
              <template #icon>
                <NIcon v-if="!loading"><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/><polyline points="17 21 17 13 7 13 7 21"/><polyline points="7 3 7 8 15 8"/></svg></NIcon>
              </template>
              保存配置
            </NButton>
          </div>
        </NCard>
      </transition>
    </div>
  </div>
</template>

<style scoped>
.slide-fade-enter-active {
  transition: all 0.3s ease-out;
}

.slide-fade-leave-active {
  transition: all 0.2s cubic-bezier(1, 0.5, 0.8, 1);
}

.slide-fade-enter-from {
  transform: translateX(20px);
  opacity: 0;
}

.slide-fade-leave-to {
  transform: translateX(-20px);
  opacity: 0;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
