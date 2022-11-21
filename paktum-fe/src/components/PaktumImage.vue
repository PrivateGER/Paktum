<template>
  <v-dialog v-model="dialog" width="80%">
    <template v-slot:activator="{ on, attrs }">
      <v-img style="width: 480px" v-bind:src="thumbnail" @click="dialog = true" v-if="!src.includes('webm')" class="expandCursor">
      </v-img>
      <video v-bind:src="src" v-if="src.includes('webm')" controls  />
      <p v-if="relatedImages.length > 0"><v-icon icon="mdi-image-multiple" /> There <span v-if="relatedImages.length === 1">is</span><span v-else>are</span> {{relatedImages.length}} related image<span v-if="relatedImages.length > 1">s</span>. Open detail view to see list.</p>
    </template>
    <v-card>
      <v-card-title class="headline">Image</v-card-title>

      <v-img v-bind:src="src" @click="dialog = false" />

      <div class="pa-4">
        <p v-if="relatedImages.length > 0">Related Images</p>
        <span class="ml-1" v-for="image in relatedImages">
          <RouterLink :to="'/image/'+image.id">
            <img :src="image.thumbnail" alt="related image">
          </RouterLink>
        </span>
      </div>

      
      <v-card-actions>
        <v-spacer />
        <v-btn color="blue darken-1" text @click="dialog = false">Close</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script>
export default {
  name: "PaktumImage",
  props: {
    src: {
      type: String,
      required: true,
    },
    thumbnail: {
      type: String,
      required: true,
    },
    relatedImages: {
      type: Array,
      required: false,
    },
  },
  data() {
    return {
      dialog: false,
    };
  },
}
</script>

<style scoped>
.expandCursor {
  cursor: zoom-in;
}
</style>