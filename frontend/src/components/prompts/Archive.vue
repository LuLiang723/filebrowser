<template>
  <div class="card floating" id="archive">
    <div class="card-title">
      <h2>{{ t('buttons.archive') }}</h2>
    </div>

    <div class="card-content">
      <p>{{ t('prompts.archiveMessage', 'Please enter a name for the archive:') }}</p>

      <input
        class="input input--block"
        type="text"
        v-model="archiveName"
        @keyup.enter="submit"
        autofocus
        :placeholder="'archive_name'"
      />

      <p style="margin-top: 1rem;">Format:</p>
      <div class="select-wrapper">
        <select class="input input--block" v-model="selectedFormat">
          <option v-for="(ext, format) in formats" :key="format" :value="format">
            .{{ ext }}
          </option>
        </select>
      </div>
    </div>

    <div class="card-action">
      <button
        class="button button--flat button--grey"
        @click="layoutStore.closeHovers"
        :disabled="isLoading"
        :aria-label="t('buttons.cancel')"
        :title="t('buttons.cancel')"
      >
        {{ t('buttons.cancel') }}
      </button>

      <button
        class="button button--flat"
        @click="submit"
        :disabled="isLoading"
        :aria-label="t('buttons.archive')"
        :title="t('buttons.archive')"
      >
        {{ isLoading ? t("processing") : t('buttons.archive') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { useLayoutStore } from "@/stores/layout";

const nameInput = ref<HTMLInputElement | null>(null);

onMounted(() => {
  nameInput.value?.focus();
  nameInput.value?.select();
});

const layoutStore = useLayoutStore();
const { t } = useI18n();

const defaultName = layoutStore.currentPrompt?.props?.defaultName || "archive";
const archiveName = ref(defaultName);

const selectedFormat = ref("zip");


const formats = {
  zip: "zip",
  tar: "tar",
  targz: "tar.gz",
  tarbz2: "tar.bz2",
  tarxz: "tar.xz",
  tarlz4: "tar.lz4",
  tarsz: "tar.sz",
  tarbr: "tar.br",
  tarzst: "tar.zst",
};

const isLoading = ref(false);

const submit = async () => {
  isLoading.value = true; 
  
  try {
    await layoutStore.currentPrompt!.confirm({
      name: archiveName.value,
      format: selectedFormat.value,
      extension: formats[selectedFormat.value as keyof typeof formats]
    });
  } finally {
    isLoading.value = false;
  }
};


</script>

<style scoped>
.select-wrapper {
  position: relative;
}
</style>