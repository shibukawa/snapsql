---
layout: page
---

<script setup>
import { onMounted } from 'vue'

onMounted(() => {
  // ルートページから /ja/ にリダイレクト
  if (typeof window !== 'undefined') {
    window.location.href = '/ja/'
  }
})
</script>

# リダイレクト中...

このページは自動的に日本語版にリダイレクトされます。

<a href="/ja/">日本語版はこちら</a>