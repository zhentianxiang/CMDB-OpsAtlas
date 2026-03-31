<script setup lang="ts">
import { reactive, ref } from "vue";
import { message } from "@/utils/message";
import { updateMyPassword } from "@/api/user";
import type { FormInstance, FormRules } from "element-plus";
import { deviceDetection } from "@pureadmin/utils";

defineOptions({
  name: "PasswordForm"
});

const formRef = ref<FormInstance>();
const loading = ref(false);
const form = reactive({
  oldPassword: "",
  newPassword: "",
  confirmPassword: ""
});

const rules = reactive<FormRules<typeof form>>({
  oldPassword: [{ required: true, message: "请输入当前密码", trigger: "blur" }],
  newPassword: [
    { required: true, message: "请输入新密码", trigger: "blur" },
    { min: 6, message: "新密码至少 6 位", trigger: "blur" }
  ],
  confirmPassword: [
    { required: true, message: "请再次输入新密码", trigger: "blur" },
    {
      validator: (_rule, value, callback) => {
        if (value !== form.newPassword) {
          callback(new Error("两次输入的新密码不一致"));
          return;
        }
        callback();
      },
      trigger: "blur"
    }
  ]
});

async function onSubmit() {
  if (!formRef.value) return;
  const valid = await formRef.value.validate().catch(() => false);
  if (!valid) return;

  loading.value = true;
  try {
    const { code, message: text } = await updateMyPassword({ ...form });
    if (code !== 0) {
      message(text || "修改密码失败", { type: "error" });
      return;
    }

    form.oldPassword = "";
    form.newPassword = "";
    form.confirmPassword = "";
    formRef.value.resetFields();
    message("密码修改成功", { type: "success" });
  } catch (error) {
    message(error?.message || "修改密码失败", { type: "error" });
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <div :class="['min-w-45', deviceDetection() ? 'max-w-full' : 'max-w-[70%]']">
    <h3 class="my-8!">修改密码</h3>
    <el-form
      ref="formRef"
      label-position="top"
      :rules="rules"
      :model="form"
      status-icon
    >
      <el-form-item label="当前密码" prop="oldPassword">
        <el-input
          v-model="form.oldPassword"
          type="password"
          show-password
          placeholder="请输入当前密码"
        />
      </el-form-item>
      <el-form-item label="新密码" prop="newPassword">
        <el-input
          v-model="form.newPassword"
          type="password"
          show-password
          placeholder="请输入新密码"
        />
      </el-form-item>
      <el-form-item label="确认新密码" prop="confirmPassword">
        <el-input
          v-model="form.confirmPassword"
          type="password"
          show-password
          placeholder="请再次输入新密码"
        />
      </el-form-item>
      <el-button type="primary" :loading="loading" @click="onSubmit">
        保存新密码
      </el-button>
    </el-form>
  </div>
</template>
