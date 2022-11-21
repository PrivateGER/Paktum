import { createRouter, createWebHistory } from 'vue-router'
import HelloWorld from '../components/HelloWorld.vue'
import RandomImagePage from "../components/RandomImagePage.vue";
import DetailedImage from "../components/DetailedImage.vue";

const routes = [
    {
        path: '/',
        name: 'Home',
        component: HelloWorld,
    },
    {
        path: '/random',
        name: 'Random Image',
        component: RandomImagePage
    },
    {
        path: '/image/:id',
        name: 'Detailed Image',
        component: DetailedImage
    }
]
const router = createRouter({
    history: createWebHistory(),
    routes,
})

export default router