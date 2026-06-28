<script setup lang="ts">
import { ref } from 'vue'

const file = ref<File | null>(null)
const uploaded = ref< Boolean >(false)

const onFileChange = (event: Event) => {
  const target = event.target as HTMLInputElement
  if (target.files && target.files.length > 0) {
    file.value = target.files?.[0]! 
  }
}

const handleUpload = async () => {
  
  const formData = new FormData()
  formData.append('file', file.value!)

    const res = await fetch('/upload', {
      method: 'POST',
      body: formData
    })

    if (res.status == 200) {
        uploaded.value = true
    } else {
        alert("Something went wrong and the file wasn't uploaded.")
    }
    
}
</script>

<template>
  <div>
    <form @submit.prevent="handleUpload">
        <h1>Upload a file</h1>

        <input type="file" @change="onFileChange">
        
        <input type="submit" value="Submit">
    </form>
  </div>
</template>