<script>
// This starter template is using Vue 3 <script setup> SFCs
// Check out https://vuejs.org/api/sfc-script-setup.html#script-setup
import HelloWorld from './components/HelloWorld.vue'

import {RouterView} from "vue-router";
import RandomImage from "./components/RandomImage.vue";

export default {
  name: "App",
  data() {
    return {
      menuItems: [
        {title: "Search", to: "/search", icon: "mdi-magnify"},
        {title: "Random", to: "/random", icon: "mdi-dice-3"},
      ],
    }
  },
  components: {
    RouterView,
  },
}

</script>

<template>
  <v-app dark>
    <v-toolbar>
      <v-toolbar-title>
        <router-link to="/" tag="span" style="cursor: pointer; text-decoration: none">
          Paktum Image Server
        </router-link>
      </v-toolbar-title>
      <v-spacer></v-spacer>
      <v-toolbar-items class="hidden-xs-only">
        <RouterLink v-for="item in menuItems" :to="item.to" tag="span" style="cursor: pointer; text-decoration: none">
          <v-btn class="justify-center align-center" flat>
            <v-icon>{{item.icon}}</v-icon>
            {{item.title}}
          </v-btn>
        </RouterLink>
      </v-toolbar-items>
    </v-toolbar>
    <div class="pa-8">
      <RouterView />
    </div>
  </v-app>
</template>

<style scoped>

</style>
