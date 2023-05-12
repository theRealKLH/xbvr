<template>
  <section>
    <b-field class="row" position="is-centered" v-for="(item, i) in list" :key="`item-${i}`">
      <b-input  :class="`list-editor-input list-editor-input-${type}-${i}`" :value="item" @blur="blur(i)" />
      <p class="control">
        <!--<b-button type="is-danger" @click="deleteRow(i)">Delete</b-button>-->
        <b-button type="is-light" @click="deleteRow(i)" icon-right="delete" />
      </p>
      <p class="control">
        <a v-if="showUrl" class="button is-light" 
          :title="`Go to $(${item}`" :href="item" target="_blank" rel="noreferrer">
          <b-icon pack="mdi" icon="link" size="is-small" />
        </a>
      </p>
    </b-field>

    <b-field>
      <b-button class="control" type="is-info" icon-right="plus-circle-outline" @click="addRow">{{$t('Add item')}}</b-button>
    </b-field>
  </section>
</template>

<script>
export default {
  name: 'ListEditor',
  props: {
    list: Array,
    type: String,
    blurFn: Function,
    showUrl: Boolean,
  },
  methods: {
    addRow () {
      this.list.push('')
    },
    deleteRow (i) {
      this.list.splice(i, 1)
    },
    blur (i) {
      this.list[i] = document.querySelector(`.list-editor-input-${this.type}-${i} input`).value
      this.blurFn.call(null)
    }
  }
}
</script>

<style scoped>
  .list-editor-input {
    width: 100%;
  }
</style>
