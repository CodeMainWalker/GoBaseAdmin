<!-- 登录页面 -->
<template>
  <div class="auth-page">
    <!-- 浅底 + 浮动圆装饰 -->
    <div class="auth-bg" aria-hidden="true">
      <span class="auth-bg__circle auth-bg__circle--1"></span>
      <span class="auth-bg__circle auth-bg__circle--2"></span>
      <span class="auth-bg__circle auth-bg__circle--3"></span>
      <span class="auth-bg__circle auth-bg__circle--4"></span>
    </div>

    <AuthTopBar plain />

    <!-- 左上角 Logo + 系统名 -->
    <header class="auth-header">
      <ArtLogo :size="45" />
      <h1 class="auth-header__name">{{ systemName }}</h1>
    </header>

    <main class="auth-main">
      <section class="auth-card">
        <!-- 左半区：品牌 + 插画 + 能力条 -->
        <div class="auth-card__brand">
          <p class="auth-card__brand-title">{{ systemName }} v{{ version }}</p>
          <p class="auth-card__slogan">---- {{ $t('login.slogan') }}</p>

          <div class="auth-card__illustration">
            <ThemeSvg :src="loginIcon" size="100%" />
          </div>

          <ul class="auth-card__capabilities">
            <li v-for="key in capabilityKeys" :key="key">{{ $t(key) }}</li>
          </ul>
        </div>

        <!-- 右半区：表单 -->
        <div class="auth-card__form">
          <h2 class="auth-card__title">{{ $t('login.title') }}</h2>

          <ElForm
            ref="formRef"
            class="auth-form"
            :model="formData"
            :rules="rules"
            :key="formKey"
            @keyup.enter="handleSubmit"
          >
            <ElFormItem prop="username">
              <ElInput
                v-model.trim="formData.username"
                class="auth-field"
                :placeholder="$t('login.placeholder.username')"
                autocomplete="off"
              >
                <template #prefix>
                  <ElIcon><User /></ElIcon>
                </template>
              </ElInput>
            </ElFormItem>

            <ElFormItem prop="password">
              <ElInput
                v-model.trim="formData.password"
                class="auth-field"
                :placeholder="$t('login.placeholder.password')"
                type="password"
                autocomplete="off"
                show-password
              >
                <template #prefix>
                  <ElIcon><Lock /></ElIcon>
                </template>
              </ElInput>
            </ElFormItem>

            <ElFormItem prop="code">
              <div class="auth-captcha">
                <ElInput
                  v-model.trim="formData.code"
                  class="auth-field auth-captcha__input"
                  :placeholder="$t('login.placeholder.code')"
                  autocomplete="off"
                >
                  <template #prefix>
                    <ElIcon><Key /></ElIcon>
                  </template>
                </ElInput>
                <!-- 原生 button：鼠标、键盘（Tab + 回车/空格）都能刷新验证码 -->
                <button
                  type="button"
                  class="auth-captcha__btn"
                  :title="$t('login.refreshCaptcha')"
                  :aria-label="$t('login.refreshCaptcha')"
                  @click="refreshCaptcha"
                >
                  <img :src="captcha" alt="" />
                </button>
              </div>
            </ElFormItem>

            <div class="auth-options">
              <ElCheckbox v-model="formData.rememberPassword">
                {{ $t('login.rememberPwd') }}
              </ElCheckbox>
            </div>

            <ElButton
              class="auth-submit"
              type="primary"
              :loading="loading"
              v-ripple
              @click="handleSubmit"
            >
              {{ $t('login.btnText') }}
            </ElButton>
          </ElForm>
        </div>
      </section>
    </main>

    <!-- 底部信息栏：安全提示 + 版权/版本 -->
    <footer class="auth-footer">
      <p class="auth-footer__hint">
        <ElIcon><Monitor /></ElIcon>
        <span>{{ $t('login.securityHint') }}</span>
      </p>
      <p class="auth-footer__copy">© {{ year }} {{ systemName }} · v{{ version }}</p>
    </footer>
  </div>
</template>

<script setup lang="ts">
  import AppConfig from '@/config'
  import { useUserStore } from '@/store/modules/user'
  import { useI18n } from 'vue-i18n'
  import { HttpError } from '@/utils/http/error'
  import { fetchCaptcha, fetchLogin, fetchGetUserInfo } from '@/api/auth'
  import { ElNotification, type FormInstance, type FormRules } from 'element-plus'
  import { Key, Lock, Monitor, User } from '@element-plus/icons-vue'
  import loginIcon from '@imgs/svg/login_icon.svg'

  defineOptions({ name: 'Login' })

  const { t, locale } = useI18n()
  const formKey = ref(0)

  // 监听语言切换，重置表单
  watch(locale, () => {
    formKey.value++
  })

  const userStore = useUserStore()
  const router = useRouter()

  const captcha = ref(
    'data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7'
  )

  const systemName = AppConfig.systemInfo.name
  const year = new Date().getFullYear()
  // .vue 里没有 __APP_VERSION__ 的 eslint 全局声明（那个常量只有 .ts 工具在用），
  // 这里用同为 VITE_VERSION 来源的 import.meta.env，避免新引入一条 lint 报错。
  const version = import.meta.env.VITE_VERSION as string
  const formRef = ref<FormInstance>()

  const capabilityKeys = [
    'login.capability.rbac',
    'login.capability.monitor',
    'login.capability.i18n',
    'login.capability.theme'
  ]

  const formData = reactive({
    username: '',
    password: '',
    code: '',
    uuid: '',
    rememberPassword: true
  })

  const rules = computed<FormRules>(() => ({
    username: [{ required: true, message: t('login.placeholder.username'), trigger: 'blur' }],
    password: [{ required: true, message: t('login.placeholder.password'), trigger: 'blur' }],
    code: [{ required: true, message: t('login.placeholder.code'), trigger: 'blur' }]
  }))

  const loading = ref(false)

  onMounted(() => {
    refreshCaptcha()
  })

  // 登录
  const handleSubmit = async () => {
    if (!formRef.value) return

    try {
      // 表单验证
      const valid = await formRef.value.validate()
      if (!valid) return

      loading.value = true

      // 登录请求
      const { access_token, refresh_token } = await fetchLogin({
        username: formData.username,
        password: formData.password,
        code: formData.code,
        uuid: formData.uuid
      })

      // 验证token
      if (!access_token) {
        throw new Error('Login failed - no token received')
      }

      // 存储token和用户信息
      userStore.setToken(access_token, refresh_token)
      const userInfo = await fetchGetUserInfo()
      userStore.setUserInfo(userInfo)
      userStore.setLoginStatus(true)

      // 登录成功处理
      showLoginSuccessNotice()
      router.push('/')
    } catch (error) {
      // 处理 HttpError
      if (error instanceof HttpError) {
        // console.log(error.code)
      } else {
        // 处理非 HttpError
        // ElMessage.error('登录失败，请稍后重试')
        console.error('[Login] Unexpected error:', error)
      }
    } finally {
      refreshCaptcha()
      loading.value = false
    }
  }

  // 获取验证码
  // 失败提示由 utils/http 统一弹出（这里只兜住 promise，避免控制台多一条 unhandled rejection）
  const refreshCaptcha = async () => {
    fetchCaptcha()
      .then((res) => {
        formData.uuid = res.uuid
        captcha.value = res.image
      })
      .catch(() => {
        formData.uuid = ''
      })
  }

  // 登录成功提示
  const showLoginSuccessNotice = () => {
    setTimeout(() => {
      ElNotification({
        title: t('login.success.title'),
        type: 'success',
        duration: 2500,
        zIndex: 10000,
        message: `${t('login.success.message')}, ${systemName}!`
      })
    }, 150)
  }
</script>

<style lang="scss" scoped>
  /**
   * 登录页样式写在这里、不放进 ./style.css：
   * 那个文件被 register / forget-password 共用（两页仍是旧的左右分栏），改它就会连带影响那两页。
   *
   * 视觉参考 D:\mini_work\erp\GoERP\web\src\views\auth\login\index.vue：
   * 浅底 + 浮动圆 + 左上 Logo、居中 950×500 白卡（左品牌区 + 插画 / 右表单）、灰底填充输入框、全宽主色按钮。
   * 登录页固定使用自己的品牌色，不跟随用户的主题色设置 —— 否则同一张设计图在不同账号机器上会变色。
   */
  .auth-page {
    --auth-brand: #3f7df6;
    --auth-brand-hover: #79a4f9;
    --auth-brand-active: #3264c5;
    --auth-brand-panel: linear-gradient(180deg, #f7fbff 0%, #eef5ff 100%);

    // 覆盖 Element Plus 主色（只作用于登录页内部：按钮 / 复选框 / 输入聚焦态）
    --el-color-primary: var(--auth-brand);
    --el-color-primary-light-3: var(--auth-brand-hover);
    --el-color-primary-light-5: #9fbefb;
    --el-color-primary-light-7: #c5d8fc;
    --el-color-primary-light-8: #d9e5fd;
    --el-color-primary-light-9: #ecf2fe;
    --el-color-primary-dark-2: var(--auth-brand-active);
    // 底座在 :root 上把 --theme-color/--main-color 解析成了「用户主题色」，
    // 而暗色下 .dark .el-button--primary 用的是 --theme-color —— 不在这里一起覆盖，
    // 暗色登录按钮会跟着设置里的主题色变（实测用户改成绿色时按钮就变绿了）。
    --main-color: var(--auth-brand);
    --theme-color: var(--auth-brand);
    --theme-color-hover: var(--auth-brand-hover);
    --theme-color-active: var(--auth-brand-active);
    --theme-color-plain-bg: rgb(63 125 246 / 8%);
    --theme-color-plain-border: rgb(63 125 246 / 30%);
    --theme-color-disabled: rgb(63 125 246 / 45%);

    position: relative;
    display: flex;
    flex-direction: column;
    min-height: 100vh;
    min-height: 100dvh;
    // rgb(145 185 233 / 55%) 叠在白底上的效果，对齐参考图
    background: #c2d9f3;
  }

  .dark .auth-page {
    --auth-brand: #5b8ff8;
    --auth-brand-hover: #7ea7fa;
    --auth-brand-active: #4a72c6;
    --auth-brand-panel: linear-gradient(180deg, #1b2028 0%, #171b22 100%);
    --el-color-primary-light-5: #a4c0fb;
    --el-color-primary-light-7: #c9d9fd;
    --el-color-primary-light-8: #dde7fe;
    --el-color-primary-light-9: #eef3fe;
    --theme-color-plain-bg: rgb(91 143 248 / 12%);
    --theme-color-plain-border: rgb(91 143 248 / 30%);
    --theme-color-disabled: rgb(91 143 248 / 45%);

    background: #1e232b;
  }

  /* 浮动装饰圆（固定层裁切，不影响页面滚动） */
  .auth-bg {
    position: fixed;
    inset: 0;
    z-index: 0;
    overflow: hidden;
    pointer-events: none;
  }

  .auth-bg__circle {
    position: absolute;
    background: linear-gradient(to right, rgb(63 125 246 / 8%), rgb(63 125 246 / 4%));
    border-radius: 50%;
    animation: authFloat 3s ease-in-out infinite;
  }

  .auth-bg__circle--1 {
    top: 100px;
    left: 40px;
    width: 100px;
    height: 100px;
    animation-duration: 2.5s;
  }

  .auth-bg__circle--2 {
    bottom: 5%;
    left: 15%;
    width: 150px;
    height: 150px;
  }

  .auth-bg__circle--3 {
    top: 90px;
    right: 12%;
    width: 145px;
    height: 145px;
    animation-duration: 2.5s;
  }

  .auth-bg__circle--4 {
    top: 60%;
    right: 5%;
    width: 160px;
    height: 160px;
    animation-duration: 3.5s;
  }

  .dark .auth-bg__circle {
    background: linear-gradient(to right, rgb(91 143 248 / 14%), rgb(91 143 248 / 6%));
  }

  @keyframes authFloat {
    0% {
      transform: translateY(0) scale(1);
    }

    50% {
      transform: translateY(25px) scale(1.1);
    }

    100% {
      transform: translateY(0) scale(1);
    }
  }

  /* 左上角品牌 */
  .auth-header {
    position: relative;
    z-index: 2;
    display: flex;
    gap: 10px;
    align-items: center;
    height: 80px;
    padding: 0 20px;
  }

  .auth-header__name {
    font-size: 20px;
    font-weight: 700;
    color: var(--art-gray-900);
  }

  /* 卡片区 */
  .auth-main {
    position: relative;
    z-index: 2;
    display: flex;
    flex: 1;
    align-items: center;
    justify-content: center;
    padding: 8px 16px 24px;
  }

  .auth-card {
    display: flex;
    // margin: auto 0（而不是靠容器 align-items）——内容高于视口时自动边距归零，不会被裁掉
    width: min(950px, 100%);
    min-height: 500px;
    margin: auto 0;
    overflow: hidden;
    background: var(--default-box-color);
    border-radius: 12px;
    box-shadow: 0 2px 4px 2px rgb(0 0 0 / 8%);
  }

  .auth-card__brand {
    display: flex;
    flex: none;
    flex-direction: column;
    width: 50%;
    padding: 24px 28px 20px;
    background: var(--auth-brand-panel);
    border-right: 1px solid rgb(63 125 246 / 8%);
  }

  .auth-card__brand-title {
    font-size: 26px;
    font-weight: 700;
    color: var(--auth-brand);
  }

  .auth-card__slogan {
    margin-top: 6px;
    font-size: 15px;
    color: var(--art-gray-600);
    text-align: right;
  }

  .auth-card__illustration {
    display: flex;
    flex: 1;
    align-items: center;
    justify-content: center;
    min-height: 0;
    padding: 12px 0;
  }

  .auth-card__capabilities {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    margin-top: 4px;
    list-style: none;

    li {
      font-size: 12px;
      color: var(--art-gray-600);

      & + li::before {
        margin: 0 6px;
        content: '·';
      }
    }
  }

  .auth-card__form {
    display: flex;
    flex: 1;
    flex-direction: column;
    justify-content: center;
    min-width: 0;
    padding: 32px 40px;
  }

  .auth-card__title {
    margin-bottom: 22px;
    font-size: 28px;
    font-weight: 600;
    color: var(--art-gray-900);
  }

  .auth-form {
    :deep(.el-form-item) {
      margin-bottom: 16px;
    }

    :deep(.el-form-item__error) {
      padding-top: 3px;
      font-size: 12px;
    }
  }

  /* 参考图是灰底填充式输入框（无描边，聚焦才描边） */
  .auth-field {
    width: 100%;

    --el-input-bg-color: var(--art-gray-200);
    --el-input-border-color: transparent;
    --el-input-hover-border-color: transparent;
    --el-input-focus-border-color: var(--auth-brand);
    --el-input-text-color: var(--art-gray-900);

    :deep(.el-input__wrapper) {
      min-height: 42px;
      padding: 0 14px;
      border-radius: 6px;
    }

    :deep(.el-input__inner) {
      font-size: 14px;
    }

    :deep(.el-input__prefix) {
      margin-right: 8px;
      color: var(--art-gray-500);
    }
  }

  .auth-captcha {
    display: flex;
    gap: 10px;
    align-items: center;
    width: 100%;
  }

  .auth-captcha__input {
    flex: 1;
    min-width: 0;
  }

  .auth-captcha__btn {
    display: flex;
    flex: none;
    align-items: center;
    justify-content: center;
    width: 116px;
    height: 42px;
    padding: 0;
    overflow: hidden;
    cursor: pointer;
    background: var(--default-box-color);
    border: 1px solid var(--default-border);
    border-radius: 6px;
    transition: border-color 0.2s ease;

    img {
      display: block;
      width: 100%;
      height: 100%;
      object-fit: contain;
    }

    &:hover {
      border-color: var(--auth-brand);
    }

    &:focus-visible {
      outline: 2px solid var(--auth-brand);
      outline-offset: 2px;
    }
  }

  .auth-options {
    display: flex;
    align-items: center;
    min-height: 24px;
    margin-top: -2px;
  }

  .auth-submit {
    width: 100%;
    // 底座把 --el-component-custom-height 定成 36px 并带 !important，这里必须一起 !important 才生效
    height: 46px !important;
    margin-top: 20px;
    font-size: 16px;
    font-weight: 500;
    border-radius: 6px;
  }

  /* 底部信息栏 */
  .auth-footer {
    position: relative;
    z-index: 2;
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
    padding: 12px 24px;
    font-size: 12px;
    color: var(--art-gray-600);
    background: rgb(255 255 255 / 65%);
    backdrop-filter: blur(4px);
    border-top: 1px solid rgb(255 255 255 / 70%);
  }

  .dark .auth-footer {
    background: rgb(22 22 24 / 72%);
    border-top-color: var(--default-border);
  }

  .auth-footer__hint {
    display: flex;
    gap: 6px;
    align-items: center;
  }

  /* ---------------- 响应式 ---------------- */
  @media (width <= 991px) {
    .auth-header {
      height: 64px;
      padding: 0 16px;
    }

    .auth-card {
      width: min(520px, 100%);
      min-height: 0;
    }

    .auth-card__brand {
      display: none;
    }

    .auth-card__form {
      padding: 28px 24px;
    }
  }

  @media (width <= 480px) {
    .auth-main {
      padding: 4px 14px 20px;
    }

    .auth-card__form {
      padding: 24px 18px;
    }

    .auth-card__title {
      margin-bottom: 18px;
      font-size: 24px;
    }

    .auth-captcha__btn {
      width: 96px;
    }

    .auth-footer {
      flex-direction: column;
      gap: 6px;
      text-align: center;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .auth-bg__circle {
      animation: none;
    }
  }
</style>
