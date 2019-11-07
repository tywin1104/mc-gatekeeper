import axios from 'axios';

const API_HOST = process.env.REACT_APP_API_HOST ? process.env.REACT_APP_API_HOST : "";

class RecaptchaService {

    verify(recapchaToken) {
        return axios.post(`${API_HOST}/api/v1/recaptcha/verify`, {recapchaToken})
    }
}

export default new RecaptchaService();