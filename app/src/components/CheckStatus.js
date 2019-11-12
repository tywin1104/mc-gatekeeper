import React from "react";
import { ListGroup, ListGroupItem, Container, Button } from 'reactstrap';
import moment from 'moment'
import RequestsService from '../service/RequestsService'
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
    const { match: { params } } = this.props;
    RequestsService.getRequestByEncodedID(params.id)
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

  getButtonColor(status)  {
      switch(status) {
          case "Approved":
              return "success"
          case "Denied":
              return "danger"
          default:
              return "info"
      }
  }

  render() {
      let display
      let currentRequest = this.state.currentRequest
      if(!this.state.invalid && currentRequest) {
          display = (
        <Container>
        <ListGroup>
            <ListGroupItem active tag="a" href="#" action>Your Application Status</ListGroupItem>
            <ListGroupItem tag="a" action><strong>Username  </strong> {currentRequest.username}</ListGroupItem>
            <ListGroupItem tag="a"  action><strong>Email  </strong> {currentRequest.email}</ListGroupItem>
            <ListGroupItem tag="a" action>
                <strong>Status  </strong>
                <Button
                    color={this.getButtonColor(currentRequest.status)} type="button">{currentRequest.status}
                </Button>
            </ListGroupItem>
            <ListGroupItem tag="a"  action><strong>Reference ID  </strong> {currentRequest._id}</ListGroupItem>
            <ListGroupItem disabled tag="a" href="#" action>
                <p>Application submitted { moment.parseZone(currentRequest.timestamp).local().fromNow()}</p>
                <p>If you haven't heard from us within 24 hours, please contact us with your application ID above for reference</p>
            </ListGroupItem>
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

