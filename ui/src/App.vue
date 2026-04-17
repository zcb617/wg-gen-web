<template>
  <v-app id="inspire">
    <Notification v-bind:notification="notification"/>
    <div v-if="this.isAuthenticated">
      <Header/>

      <v-main>
        <v-container>
          <router-view />
        </v-container>
      </v-main>

      <Footer/>
    </div>
    <div v-else-if="this.showLoginForm" class="login-container">
      <v-card class="login-card" max-width="400">
        <v-card-title class="headline">WireGuard Login</v-card-title>
        <v-card-text>
          <v-form ref="loginForm" v-model="loginValid">
            <v-text-field
              v-model="loginUsername"
              label="Username"
              :rules="[v => !!v || 'Username is required']"
              required
            />
            <v-text-field
              v-model="loginPassword"
              label="Password"
              type="password"
              :rules="[v => !!v || 'Password is required']"
              required
              @keyup.enter="doLogin"
            />
          </v-form>
        </v-card-text>
        <v-card-actions>
          <v-spacer/>
          <v-btn
            color="success"
            :disabled="!loginValid"
            @click="doLogin"
          >
            Login
          </v-btn>
        </v-card-actions>
      </v-card>
    </div>
  </v-app>
</template>

<script>
  import Notification from './components/Notification'
  import Header from "./components/Header";
  import Footer from "./components/Footer";
  import {mapActions, mapGetters} from "vuex";

  export default {
    name: 'App',

    components: {
      Footer,
      Header,
      Notification
    },

    data: () => ({
      notification: {
        show: false,
        color: '',
        text: '',
      },
      loginValid: false,
      loginUsername: '',
      loginPassword: '',
    }),

    computed:{
      ...mapGetters({
        isAuthenticated: 'auth/isAuthenticated',
        authStatus: 'auth/authStatus',
        authRedirectUrl: 'auth/authRedirectUrl',
        authError: 'auth/error',
        clientError: 'client/error',
        serverError: 'server/error',
      }),
      showLoginForm() {
        return this.authStatus === 'login';
      }
    },

    created () {
      this.$vuetify.theme.dark = true
    },

    mounted() {
      if (this.$route.query.code && this.$route.query.state) {
        this.oauth2_exchange({
          code: this.$route.query.code,
          state: this.$route.query.state
        })
      } else {
        this.oauth2_url()
      }
    },

    watch: {
      authError(newValue, oldValue) {
        console.log(newValue)
        this.notify('error', newValue);
      },

      clientError(newValue, oldValue) {
        console.log(newValue)
        this.notify('error', newValue);
      },

      serverError(newValue, oldValue) {
        console.log(newValue)
        this.notify('error', newValue);
      },

      isAuthenticated(newValue, oldValue) {
        console.log(`Updating isAuthenticated from ${oldValue} to ${newValue}`);
        if (newValue === true) {
          this.$router.push('/clients')
        }
      },

      authStatus(newValue, oldValue) {
        console.log(`Updating authStatus from ${oldValue} to ${newValue}`);
        if (newValue === 'redirect') {
          window.location.replace(this.authRedirectUrl)
        }
      },
    },

    methods: {
      ...mapActions('auth', {
        oauth2_exchange: 'oauth2_exchange',
        oauth2_url: 'oauth2_url',
        login: 'login',
      }),

      notify(color, msg) {
        this.notification.show = true;
        this.notification.color = color;
        this.notification.text = msg;
      },

      doLogin() {
        this.login({
          username: this.loginUsername,
          password: this.loginPassword,
        });
      }
    }
  };
</script>

<style>
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100vh;
}
.login-card {
  width: 100%;
}
</style>
