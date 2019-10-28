import axios from 'axios';

const API_BASE_URL = process.env.REACT_APP_API_BASE_URL;

class AuthService {

    login(credentials){
        return axios.post(`${API_BASE_URL}/api/v1/auth/`, credentials);
    }

    getAuthHeader() {
        return {headers: {Authorization: 'Bearer ' + this.getTokenInfo().value }};
    }

    getTokenInfo(){
        return JSON.parse(localStorage.getItem("token"));
    }


    // logOut() {
    //     localStorage.removeItem("userInfo");
    //     return axios.post(USER_API_BASE_URL + 'logout', {}, this.getAuthHeader());
    // }
}

export default new AuthService();