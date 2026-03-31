<script setup lang="ts">
import { message } from "@/utils/message";
import { h, onMounted, reactive, ref } from "vue";
import { type UserInfo, getMine, updateMine, uploadMyAvatar } from "@/api/user";
import { addDialog } from "@/components/ReDialog";
import ReCropperPreview from "@/components/ReCropperPreview";
import userAvatar from "@/assets/user.jpg";
import { useUserStoreHook } from "@/store/modules/user";
import { type DataInfo, userKey } from "@/utils/auth";
import type { FormInstance, FormRules } from "element-plus";
import { deviceDetection, storageLocal } from "@pureadmin/utils";

defineOptions({
  name: "Profile"
});

const emit = defineEmits<{
  updated: [user: UserInfo];
}>();

const userInfoFormRef = ref<FormInstance>();
const avatarInfo = ref();
const cropRef = ref();

const userInfos = reactive({
  avatar: "",
  username: "",
  nickname: "",
  email: "",
  phone: "",
  description: "",
  sex: 0,
  deptId: 0,
  dept: { name: "" }
});

const rules = reactive<FormRules<UserInfo>>({
  nickname: [{ required: true, message: "昵称必填", trigger: "blur" }]
});

function queryEmail(queryString: string, cb: (items: Array<{ value: string }>) => void) {
  if (!queryString) {
    cb([]);
    return;
  }

  const domains = ["qq.com", "163.com", "gmail.com", "outlook.com"];
  const normalized = queryString.includes("@") ? queryString.split("@")[0] : queryString;
  cb(domains.map(domain => ({ value: `${normalized}@${domain}` })));
}

function syncAvatar(avatar: string) {
  userInfos.avatar = avatar;
  useUserStoreHook().SET_AVATAR(avatar);
  const cached = storageLocal().getItem<DataInfo<number>>(userKey);
  if (cached) {
    storageLocal().setItem(userKey, {
      ...cached,
      avatar
    });
  }
}

function handleAvatarUpload() {
  addDialog({
    title: "裁剪、上传头像",
    width: "40%",
    closeOnClickModal: false,
    fullscreen: deviceDetection(),
    contentRenderer: () =>
      h(ReCropperPreview, {
        ref: cropRef,
        imgSrc: userInfos.avatar || userAvatar,
        onCropper: info => (avatarInfo.value = info)
      }),
    beforeSure: async done => {
      if (!avatarInfo.value?.blob) {
        message("请先选择并裁剪头像", { type: "warning" });
        return;
      }

      try {
        const res = await uploadMyAvatar(avatarInfo.value.blob);
        if (res.code !== 0 || !res.data?.avatar) {
          message(res.message || "头像上传失败", { type: "error" });
          return;
        }

        syncAvatar(res.data.avatar);
        emit("updated", { ...userInfos });
        message("头像上传成功", { type: "success" });
        done();
        avatarInfo.value = null;
      } catch (error: any) {
        message(error?.message || "头像上传失败", { type: "error" });
      }
    },
    closeCallBack: () => cropRef.value?.hidePopover?.()
  });
}

// 更新信息
const onSubmit = async (formEl: FormInstance) => {
  await formEl.validate((valid, fields) => {
    if (valid) {
      updateMine({
        avatar: userInfos.avatar,
        nickname: userInfos.nickname,
        email: userInfos.email,
        phone: userInfos.phone,
        description: userInfos.description,
        sex: userInfos.sex,
        deptId: userInfos.deptId
      })
        .then(({ code, message: text, data }) => {
          if (code !== 0) {
            message(text || "更新信息失败", { type: "error" });
            return;
          }
          Object.assign(userInfos, data);
          syncAvatar(data.avatar);
          emit("updated", { ...userInfos });
          message("更新信息成功", { type: "success" });
        })
        .catch(error => {
          message(error?.message || "更新信息失败", { type: "error" });
        });
    } else {
      console.log("error submit!", fields);
    }
  });
};

onMounted(async () => {
  const { code, data } = await getMine();
  if (code === 0) {
    Object.assign(userInfos, data);
    syncAvatar(data.avatar);
  }
});
</script>

<template>
  <div :class="['min-w-45', deviceDetection() ? 'max-w-full' : 'max-w-[70%]']">
    <h3 class="my-8!">个人信息</h3>
    <el-form
      ref="userInfoFormRef"
      label-position="top"
      :rules="rules"
      :model="userInfos"
    >
      <el-form-item label="头像">
        <div class="flex items-center gap-4">
          <el-avatar :size="72" :src="userInfos.avatar || userAvatar" />
          <div class="flex flex-col gap-2">
            <el-button type="primary" plain @click="handleAvatarUpload">
              上传头像
            </el-button>
            <span class="text-sm text-[var(--el-text-color-secondary)]">
              支持 PNG、JPG、WEBP、GIF，大小不超过 2MB
            </span>
          </div>
        </div>
      </el-form-item>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-x-6">
        <el-form-item label="用户名">
          <el-input v-model="userInfos.username" disabled />
        </el-form-item>
        <el-form-item label="昵称" prop="nickname">
          <el-input v-model="userInfos.nickname" placeholder="请输入昵称" />
        </el-form-item>
        <el-form-item label="所属部门">
          <el-input :model-value="userInfos.dept?.name || '-'" disabled />
        </el-form-item>
        <el-form-item label="性别">
          <el-select v-model="userInfos.sex" class="w-full">
            <el-option label="男" :value="0" />
            <el-option label="女" :value="1" />
          </el-select>
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-autocomplete
            v-model="userInfos.email"
            :fetch-suggestions="queryEmail"
            :trigger-on-focus="false"
            placeholder="请输入邮箱"
            clearable
            class="w-full"
          />
        </el-form-item>
        <el-form-item label="联系电话">
          <el-input
            v-model="userInfos.phone"
            placeholder="请输入联系电话"
            clearable
          />
        </el-form-item>
      </div>
      <el-form-item label="简介 (对应用户管理备注)">
        <el-input
          v-model="userInfos.description"
          placeholder="请输入简介/备注"
          type="textarea"
          :autosize="{ minRows: 4, maxRows: 6 }"
          maxlength="255"
          show-word-limit
        />
      </el-form-item>
      <el-button type="primary" @click="onSubmit(userInfoFormRef)">
        更新信息
      </el-button>
    </el-form>
  </div>
</template>
