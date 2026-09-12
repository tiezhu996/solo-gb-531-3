import { createApp } from 'vue'
import { createPinia } from 'pinia'
import {
  ElAlert,
  ElButton,
  ElDatePicker,
  ElDialog,
  ElDrawer,
  ElForm,
  ElFormItem,
  ElInput,
  ElInputNumber,
  ElLoading,
  ElOption,
  ElSelect,
  ElTable,
  ElTableColumn,
  ElTooltip,
} from 'element-plus'
import 'element-plus/dist/index.css'
import App from './App.vue'
import router from './router'
import './styles.css'

const app = createApp(App).use(createPinia()).use(router).use(ElLoading)
for (const component of [ElAlert, ElButton, ElDatePicker, ElDialog, ElDrawer, ElForm, ElFormItem, ElInput, ElInputNumber, ElOption, ElSelect, ElTable, ElTableColumn, ElTooltip]) {
  app.component(component.name!, component)
}
app.mount('#app')
