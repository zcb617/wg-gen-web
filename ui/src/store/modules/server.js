import ApiService from "../../services/api.service";

const state = {
  error: null,
  server: null,
  config: '',
  version: '_ci_build_not_run_properly_',
  dnscrypt: null,
}

const getters = {
  error(state) {
    return state.error;
  },

  server(state) {
    return state.server;
  },

  version(state) {
    return state.version;
  },

  config(state) {
    return state.config;
  },

  dnscrypt(state) {
    return state.dnscrypt;
  },
}

const actions = {
  error({ commit }, error){
    commit('error', error)
  },

  read({ commit, dispatch }){
    ApiService.get("/server")
      .then(resp => {
        commit('server', resp)
        dispatch('config')
      })
      .catch(err => {
        commit('error', err)
      })
  },

  update({ commit }, server){
    ApiService.patch(`/server`, server)
      .then(resp => {
        commit('server', resp)
      })
      .catch(err => {
        commit('error', err)
      })
  },

  config({ commit }){
    ApiService.getWithConfig("/server/config", {responseType: 'arraybuffer'})
      .then(resp => {
        commit('config', resp)
      })
      .catch(err => {
        commit('error', err)
      })
  },

  version({ commit }){
    ApiService.get("/server/version")
      .then(resp => {
        commit('version', resp.version)
      })
      .catch(err => {
        commit('error', err)
      })
  },

  dnscrypt({ commit }){
    ApiService.get("/server/dnscrypt")
      .then(resp => {
        commit('dnscrypt', resp)
      })
      .catch(err => {
        commit('dnscrypt', { enabled: false })
      })
  },

}

const mutations = {
  error(state, error) {
    state.error = error;
  },

  server(state, server){
    state.server = server
  },

  config(state, config){
    state.config = config
  },

  version(state, version){
    state.version = version
  },

  dnscrypt(state, dnscrypt){
    state.dnscrypt = dnscrypt
  },
}

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations
}
