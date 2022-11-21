<template>
  <div>
    <v-btn color="green" @click="getRandomImage">Another one</v-btn>
    <br />
    <PaktumImage v-if="!imageLoading" :src="imageSrc" :thumbnail="imagePlaceholder" :related-images="relatedImages" />
  </div>
</template>

<script>
import {randomImage} from "../API";
import PaktumImage from "./PaktumImage.vue";

export default {
  name: "RandomImage",
  components: {PaktumImage},
  data() {
    return {
      imageSrc: "",
      imagePlaceholder: "",
      imageLoading: true,
      relatedImages: [],
    };
  },
  methods: {
    getRandomImage() {
      this.imageLoading = true;
      randomImage()
        .then((response) => {
          this.imageSrc = response.Url;
          this.imagePlaceholder = response.ThumbnailUrl;
          this.relatedImages = [];
          response.Related.forEach((related) => {
            this.relatedImages.push({
              src: related.Url,
              thumbnail: related.ThumbnailUrl,
              id: related.ID,
            });
          });
          this.imageLoading = false;
        })
        .catch((error) => {
          console.log(error);
        });
    },
  },
  mounted() {
    this.getRandomImage();
  },
}
</script>

<style scoped>

</style>