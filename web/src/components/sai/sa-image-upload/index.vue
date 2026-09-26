<!-- 图片上传组件 -->
<template>
  <div class="sa-image-upload">
    <div class="image-list">
      <!-- 已上传/上传中的图片 -->
      <div v-for="item in items" :key="item.uid" class="image-item" :class="{ 'is-round': round }">
        <div v-if="item.status === 'uploading'" class="item-loading">
          <el-icon class="is-loading"><Loading /></el-icon>
        </div>
        <template v-else>
          <img class="item-thumb" :src="item.url" alt="" />
          <div class="item-actions">
            <span title="预览" @click="handlePreview(item)">
              <el-icon><ZoomIn /></el-icon>
            </span>
            <span v-if="!disabled" title="删除" @click="handleRemove(item)">
              <el-icon><Delete /></el-icon>
            </span>
          </div>
        </template>
      </div>

      <!-- 上传触发器 -->
      <el-upload
        v-if="!hideUploadTrigger"
        ref="uploadRef"
        class="upload-trigger-wrap"
        list-type="text"
        :show-file-list="false"
        :multiple="multiple"
        :accept="accept"
        :disabled="disabled"
        :http-request="handleUpload"
        :before-upload="beforeUpload"
      >
        <div class="upload-trigger" :class="{ 'is-round': round }">
          <el-icon class="upload-icon"><Plus /></el-icon>
          <div v-if="showTriggerText" class="upload-text">上传图片</div>
        </div>
      </el-upload>
    </div>

    <div v-if="showTips" class="el-upload__tip">
      单个文件不超过 {{ maxSize }}MB，最多上传 {{ limit }} 张
    </div>

    <!-- 图片预览器 -->
    <el-image-viewer
      v-if="previewVisible"
      :url-list="previewUrlList"
      :initial-index="previewIndex"
      :hide-on-click-modal="true"
      :teleported="true"
      @close="handleCloseViewer"
    />
  </div>
</template>

<script lang="ts" setup>
  import { ref, computed, watch } from 'vue'
  import { Plus, ZoomIn, Delete, Loading } from '@element-plus/icons-vue'
  import { ElMessage } from 'element-plus'
  import type { UploadProps, UploadRequestOptions } from 'element-plus'
  import { uploadImage } from '@/api/auth'

  defineOptions({ name: 'SaImageUpload' })

  interface Props {
    modelValue?: string | string[] // v-model 绑定值
    multiple?: boolean // 是否支持多选
    limit?: number // 最大上传数量
    maxSize?: number // 最大文件大小(MB)
    accept?: string // 接受的文件类型
    disabled?: boolean // 是否禁用
    width?: number // 图片/上传区域宽度(px)
    height?: number // 图片/上传区域高度(px)
    round?: boolean // 是否圆形
    showTips?: boolean // 是否显示上传提示
  }

  const props = withDefaults(defineProps<Props>(), {
    modelValue: () => [],
    multiple: false,
    limit: 1,
    maxSize: 5,
    accept: 'image/*',
    disabled: false,
    width: 148,
    height: 148,
    round: false,
    showTips: true
  })

  const emit = defineEmits<{
    'update:modelValue': [value: string | string[]]
    success: [response: any]
    error: [error: any]
    change: [value: string | string[]]
  }>()

  interface ImageItem {
    uid: number
    url: string
    status: 'uploading' | 'success'
  }

  let uidSeed = 0
  const genUid = () => ++uidSeed

  const uploadRef = ref()
  const items = ref<ImageItem[]>([])
  // 已通过校验、等待进入上传流程的数量（用于并发选择时的数量限制）
  const reservedCount = ref(0)
  const previewVisible = ref(false)
  const previewIndex = ref(0)

  // 触发器文字在小尺寸下不显示
  const showTriggerText = computed(() => props.width >= 100 && props.height >= 100)

  // 当前已上传成功的地址列表
  const successUrls = computed(() =>
    items.value.filter((item) => item.status === 'success').map((item) => item.url)
  )

  const previewUrlList = computed(() => successUrls.value)

  // 达到数量上限时隐藏上传按钮
  const hideUploadTrigger = computed(
    () => props.disabled || items.value.length + reservedCount.value >= props.limit
  )

  // 归一化 modelValue
  const normalizeValue = (value?: string | string[]): string[] => {
    if (!value) return []
    const urls = Array.isArray(value) ? value : [value]
    return urls.filter((url) => typeof url === 'string' && url)
  }

  const isSameUrls = (a: string[], b: string[]) =>
    a.length === b.length && a.every((url, index) => url === b[index])

  // 外部值变化时同步到内部列表（跳过组件自身触发的更新，避免覆盖上传中的占位）
  watch(
    () => props.modelValue,
    (newVal) => {
      const urls = normalizeValue(newVal)
      if (isSameUrls(urls, successUrls.value)) return
      items.value = urls.map((url) => ({ uid: genUid(), url, status: 'success' }))
    },
    { immediate: true, deep: true }
  )

  // 上传前校验
  const beforeUpload: UploadProps['beforeUpload'] = (file) => {
    if (!file.type.startsWith('image/')) {
      ElMessage.error('只能上传图片文件!')
      return false
    }

    if (file.size / 1024 / 1024 >= props.maxSize) {
      ElMessage.error(`图片大小不能超过 ${props.maxSize}MB!`)
      return false
    }

    if (items.value.length + reservedCount.value >= props.limit) {
      ElMessage.warning(`最多只能上传 ${props.limit} 张图片，请先删除已有图片`)
      return false
    }

    reservedCount.value++
    return true
  }

  // 自定义上传：先按选择顺序占位，再回填地址，保证并发上传的顺序与结果正确
  const handleUpload = async (options: UploadRequestOptions) => {
    const { file } = options
    const item: ImageItem = { uid: genUid(), url: '', status: 'uploading' }
    items.value.push(item)
    reservedCount.value = Math.max(0, reservedCount.value - 1)

    try {
      const formData = new FormData()
      formData.append('file', file)

      const response: any = await uploadImage(formData)
      const imageUrl = response?.data?.url || response?.data || response?.url || ''

      if (!imageUrl) {
        throw new Error('上传失败，未返回图片地址')
      }

      item.url = imageUrl
      item.status = 'success'
      updateModelValue()

      emit('success', response)
      ElMessage.success('上传成功!')
    } catch (error: any) {
      removeItem(item.uid)
      emit('error', error)
      ElMessage.error(error?.message || '上传失败!')
    }
  }

  const removeItem = (uid: number) => {
    const index = items.value.findIndex((item) => item.uid === uid)
    if (index > -1) items.value.splice(index, 1)
  }

  // 删除图片
  const handleRemove = (item: ImageItem) => {
    removeItem(item.uid)
    updateModelValue()
  }

  // 预览图片
  const handlePreview = (item: ImageItem) => {
    const index = successUrls.value.indexOf(item.url)
    previewIndex.value = index > -1 ? index : 0
    previewVisible.value = true
  }

  const handleCloseViewer = () => {
    previewVisible.value = false
  }

  // 更新 v-model 值
  const updateModelValue = () => {
    const urls = [...successUrls.value]
    const value = props.multiple ? urls : urls[0] || ''
    emit('update:modelValue', value)
    emit('change', value)
  }
</script>

<style scoped lang="scss">
  .sa-image-upload {
    .image-list {
      display: flex;
      flex-flow: row wrap;
      align-items: flex-start;
      justify-content: flex-start;
      gap: 8px;
    }

    .image-item,
    .upload-trigger {
      flex: 0 0 auto;
      box-sizing: border-box;
      width: v-bind('width + "px"');
      height: v-bind('height + "px"');
    }

    .image-item {
      position: relative;
      overflow: hidden;
      border: 1px solid var(--el-border-color);
      border-radius: 6px;
      background-color: var(--el-fill-color-blank);

      &.is-round {
        border-radius: 50%;
      }
    }

    .item-thumb {
      display: block;
      width: 100%;
      height: 100%;
      object-fit: cover;
    }

    .item-loading {
      display: flex;
      align-items: center;
      justify-content: center;
      width: 100%;
      height: 100%;
      color: var(--el-color-primary);
      background-color: var(--el-fill-color-light);
    }

    .item-actions {
      position: absolute;
      inset: 0;
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 16px;
      color: #fff;
      font-size: 18px;
      cursor: pointer;
      opacity: 0;
      background-color: var(--el-overlay-color-lighter);
      transition: opacity var(--el-transition-duration);
    }

    .image-item:hover .item-actions {
      opacity: 1;
    }

    .upload-trigger-wrap {
      flex: 0 0 auto;

      :deep(.el-upload) {
        width: v-bind('width + "px"');
        height: v-bind('height + "px"');
      }
    }

    .upload-trigger {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      border: 1px dashed var(--el-border-color-darker);
      border-radius: 6px;
      cursor: pointer;
      background-color: var(--el-fill-color-lighter);
      transition: var(--el-transition-duration-fast);

      &:hover {
        border-color: var(--el-color-primary);
      }

      &.is-round {
        border-radius: 50%;
      }

      .upload-icon {
        font-size: 28px;
        color: var(--el-text-color-secondary);
      }

      .upload-text {
        margin-top: 8px;
        font-size: 14px;
        color: var(--el-text-color-regular);
      }
    }

    .el-upload__tip {
      margin-top: 7px;
      font-size: 12px;
      line-height: 1.5;
      color: var(--el-text-color-placeholder);
    }
  }
</style>
