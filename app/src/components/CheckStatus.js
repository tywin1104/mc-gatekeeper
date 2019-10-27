import React from "react";
import axios from "axios";
import { ListGroup, ListGroupItem, Container, Button } from 'reactstrap';
import moment from 'moment'
import './CheckStatus.css';

class CheckStatus extends React.Component {
  constructor(props) {
    super()
    this.state = {
        currentRequest: {},
        invalid: false
    };
  }
  componentDidMount() {
    let api_base_url = process.env.REACT_APP_API_BASE_URL
    const { match: { params } } = this.props;
    axios.get(`${api_base_url}/api/v1/requests/${params.id}`)
    .then(res => {
        if(res.status === 200) {
            this.setState({
                currentRequest : res.data.request
            })
        }
    })
    .catch(error => {
        this.setState({
            invalid: true
        })
    });

  }

  render() {
      let display
      let currentRequest = this.state.currentRequest
      if(!this.state.invalid && currentRequest) {
          display = (
        <Container>
        <ListGroup>
            <ListGroupItem active tag="a" href="#" action>Your Application Status</ListGroupItem>
            <ListGroupItem tag="a" action><strong>Minecraft Username  </strong> {currentRequest.username}</ListGroupItem>
            <ListGroupItem tag="a"  action><strong>Email  </strong> {currentRequest.email}</ListGroupItem>
            <ListGroupItem tag="a" action>
                <strong>Status  </strong>
                <Button
                    color="info" type="button">{currentRequest.status}
                </Button>
            </ListGroupItem>
            <ListGroupItem disabled tag="a" href="#" action>Application submitted { moment.parseZone(currentRequest.timestamp).local().fromNow()}</ListGroupItem>
        </ListGroup>
        </Container>
        )
      }else {
        display = (
            <h1>Invalid</h1>
        )
      }
      return (
          <div>
              {display}
          </div>
      )
  }
}


 export default CheckStatus;

