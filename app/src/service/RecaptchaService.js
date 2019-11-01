import axios from 'axios';

const API_HOST = window.REACT_APP_API_HOST ? window.REACT_APP_API_HOST : "";

class RecaptchaService {

    verify(recapchaToken) {
        return axios.post(`${API_HOST}/api/recaptcha/verify`, {recapchaToken})
    }
}

export default new RecaptchaService();