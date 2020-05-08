import Vue from "vue";
import Vuex from "vuex";

import sceneList from "./sceneList";
import messages from "./messages";
import overlay from "./overlay";
import files from "./files";
import optionsStorage from "./optionsStorage";
import optionsDLNA from "./optionsDLNA";
import optionsSites from "./optionsSites";


Vue.use(Vuex);

export default new Vuex.Store({
  modules: {
    sceneList,
    messages,
    overlay,
    files,
    optionsStorage,
    optionsDLNA,
    optionsSites,
  }
})
