
class Body extends React.Component{
  constructor(props){
    super(props)
    this.state = {
      score_suffix: ""
    }
  }
  updateSuffix(){
    this.setState({
      score_suffix :
      SessionID + "?" + Math.floor(Math.random()*10000)
    });
  }
  render(){
    return (
      <div style={ {"height":"100%"} }>
      <div className="divHdr">
      <MenuBar updateSuffix={this.updateSuffix.bind(this)} />
      </div>
      <div className="d-inline-block" style={ {"height":"97%"} }>
      <MainFrame score_suffix={this.state.score_suffix} />
      </div>
      </div>
    );
  }
}
class PreviewButton extends React.Component{
  render(){
    return (
      <div onClick={()=>{send(this.props.updateSuffix)} } className="previewButton">
      <u>プレビュー</u>
      </div>
    );
  }
}
class GithubButton extends React.Component{
  render(){
    return (
      <div className="githubButton">
      <a href="https://github.com/norisio/lycloud/" target="_blank">GitHub</a>
      </div>
    );
  }
}
class MenuBar extends React.Component{
  render(){
    return (
      <header className="hdrMenu">
      Cloud Lilypond       
      <PreviewButton updateSuffix={this.props.updateSuffix} />
      <GithubButton />
      </header>
    );
  }
}

class LeftPane extends React.Component{
  render(){
    return (
      <textarea id="score_area" className="editor" autoFocus="true"></textarea>
    );
  }
}
class RightPane extends React.Component{
  getPdfPath = ()=>{
    var sfx = this.props.score_suffix
    if (sfx === "") {
      return ""
    }else{
      return /* "./pdfjs/web/viewer.html?file=" +*/ document.location.origin + "/get-score/" + sfx
    }
  }
  render(){
    return (
      <iframe className="preview" src={this.getPdfPath()} id="previewIframe">
      </iframe>
    );
  }
}


class MainFrame extends React.Component{
  render(){
    return (
      <div className="row h-100" style={ {"height":"100%"} }>
      <div className="h-100 col-sm-5" style={ {"height":"100%"} }>
      <LeftPane />
      </div>
      <div className="h-100 col-sm-7" style={ {"height":"100%"} }>
      <RightPane score_suffix={this.props.score_suffix} />
      </div>
      </div>
    );
  }
}

ReactDOM.render(
  <Body />,
  document.getElementById("body")
);
initialPreview();


//prevent moving focus and insert spaces
document.getElementById("score_area").addEventListener("keydown", (e)=>{
  var elem, end, start, value;
  if (e.keyCode === 9) {  //tab key
    if(e.preventDefault){
      e.preventDefault();
    }
    elem = e.target;
    start = elem.selectionStart;
    end = elem.selectionEnd;
    value = elem.value;
    const inserted = "  ";
    elem.value = "" + (value.substring(0, start)) + inserted + (value.substring(end));
    elem.selectionStart = elem.selectionEnd = start + inserted.length;
    return false;
  }
});

