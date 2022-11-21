<template>
  <h1>Details about image with ID {{ $route.params.id }}</h1>

  <v-container>
    <v-row no-gutters>
      <v-col>
        <PaktumImage v-if="!imageLoading" :src="imageData.Url" :thumbnail="imageData.ThumbnailUrl" :related-images="relatedImages" />
      </v-col>
      <v-col v-if="!imageLoading">
        <p>Filename: {{ imageData.Filename }}</p>
        <p>Tags: <RouterLink v-for="tag in imageData.Tags" :to="'/search?q=' + tag"><v-chip class="ml-1 mt-1" link color="secondary">{{ tag }}</v-chip></RouterLink></p>
        <p>PHash: {{ imageData.PHash }}</p>
        <p>Added at: {{ new Date(imageData.Added * 1000) }}</p>
        <p>Size: {{ imageData.Width }}x{{ imageData.Height }}</p>
        <p>Filesize: {{ imageData.Size }}</p>
        <p>Rating: {{ imageData.Rating }}</p>

        <p>Related images:</p>
        <v-container>
          <v-row>
            <v-col v-for="image in relatedImages" :key="image.ID">
              <RouterLink :to="'/image/'+image.id">
                <img style="max-width: 120px" :src="image.thumbnail" alt="related image">
              </RouterLink>
            </v-col>
          </v-row>
        </v-container>
      </v-col>
    </v-row>
  </v-container>


</template>

<script>
import PaktumImage from "./PaktumImage.vue";
import {imageById} from "../API";
export default {
  name: "DetailedImage",
  components: {PaktumImage},
  data() {
    return {
      imageData: {},
      imageLoading: true,
      relatedImages: [],
    };
  },
  methods: {
    getImage(id) {
      imageById(id).then((response) => {
        this.imageData = response;
        this.relatedImages = [];
        response.Related.forEach((related) => {
          this.relatedImages.push({
            src: related.Url,
            thumbnail: related.ThumbnailUrl,
            id: related.ID,
          });
        });
        this.imageLoading = false;
      });
    },
  },
  mounted() {
    this.getImage(this.$route.params.id);
  }
}
</script>

<style scoped>

</style>