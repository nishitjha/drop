<script setup lang="ts">
import { ref } from "vue";

const file = ref<File | null>(null);
const shared = ref<Boolean>(false);

const onFileChange = (event: Event) => {
  const target = event.target as HTMLInputElement;
  if (target.files && target.files.length > 0) {
    file.value = target.files?.[0]!;
  }
};

const handleShare = async () => {
  const formData = new FormData();
  formData.append("file", file.value!);

  const res = await fetch("/upload", {
    method: "POST",
    body: formData,
  });

  if (res.status == 200) {
    shared.value = true;
  } else {
    alert("Something went wrong and the file wasn't shared.");
  }
};
</script>

<template>
  <div v-if="!shared" class="share">
    <form @submit.prevent="handleShare">
      <h1>Share a file</h1>

      <input type="file" @change="onFileChange" />

      <input type="submit" value="Submit" />
    </form>
  </div>

  <div v-else="shared" class="success-share">
    Shared with Nishit's Machine.
    <button @click="shared = !shared">Share another</button>
  </div>
</template>

<style>
.success-share {
}
</style>
