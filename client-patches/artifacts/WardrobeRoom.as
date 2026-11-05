package ru.pragmatix.wormix.gui.house.wardrobe
{
   import flash.display.DisplayObject;
   import flash.events.Event;
   import flash.geom.Rectangle;
   import flash.text.TextField;
   import flash.utils.setTimeout;
   import ru.pragmatix.flox.core.SocketEvents;
   import ru.pragmatix.flox.gui.controls.menu.IUIMenuItem;
   import ru.pragmatix.flox.gui.controls.menu.UIPagingMenuWithInfo;
   import ru.pragmatix.flox.gui.core.UIMessages;
   import ru.pragmatix.flox.gui.core.UISettings;
   import ru.pragmatix.flox.gui.core.UITexts;
   import ru.pragmatix.flox.logger.Logger;
   import ru.pragmatix.flox.net.IncommingMessageEvent;
   import ru.pragmatix.flox.utils.MessageUtils;
   import ru.pragmatix.wormix.gui.Messages;
   import ru.pragmatix.wormix.gui.avatarmenu.UserHint;
   import ru.pragmatix.wormix.gui.house.IUserRoom;
   import ru.pragmatix.wormix.gui.window.Message;
   import ru.pragmatix.wormix.loader.Classes;
   import ru.pragmatix.wormix.messages.client.SelectStuff;
   import ru.pragmatix.wormix.messages.server.SelectStuffResult;
   import ru.pragmatix.wormix.model.items.HatRecord;
   import ru.pragmatix.wormix.model.items.ItemsStore;
   import ru.pragmatix.wormix.model.items.backpack.BackpackItem;
   import ru.pragmatix.wormix.model.user.UserProfile;
   import ru.pragmatix.wormix.net.MainConnection;
   import ru.pragmatix.wormix.objects.graphics.CharGraphics;
   import ru.pragmatix.wormix.settings.FontSettings;
   
   public class WardrobeRoom extends UIPagingMenuWithInfo implements IUserRoom
   {
      
      public static const WARDROBE_ID:String = "Wardrobe";
      
      private var log:Logger = new Logger(WardrobeRoom);
      
      private var p:UserProfile;
      
      private var charIcon:CharGraphics;
      
      private const nameSwf:String = "wardrobe_room";
      
      // The hat and the artifact have their own slots: both stay highlighted in the list.
      // A click on an artifact puts it on, a click on the one that is on takes it off.
      private var data:Array = [];
      
      private var chosenHat:BackpackItem;
      
      private var chosenArtifact:BackpackItem;
      
      public function WardrobeRoom()
      {
         super(null);
      }
      
      override public function initFromContainer(container:DisplayObject) : void
      {
         super.initFromContainer(container);
         container.scrollRect = new Rectangle(0,0,UISettings.stageWidth,UISettings.stageHeight);
         UITexts.embedFont(TextField(getChildByName("title")),FontSettings.TITLE_FONT);
         setTitle(Messages.WARDROBE_QTIP);
      }
      
      override protected function initParams() : void
      {
         super.initParams();
         alignByCenter = false;
         id = WARDROBE_ID;
         pageSelectionRequired = false;
      }
      
      override public function deactivate() : void
      {
         this.charIcon.stay();
         super.deactivate();
      }
      
      public function save(callback:Function) : void
      {
         // The hat first, then the artifact.
         var self:WardrobeRoom = this;
         this.saveHat(function():void
         {
            self.saveArtifact(callback);
         });
      }
      
      private function saveHat(callback:Function) : void
      {
         var item:HatRecord = null;
         if(Boolean(this.chosenHat) && this.p == Application.userProfile)
         {
            item = this.chosenHat.getRecord() as HatRecord;
            if(!item.equals(this.p.getHat()))
            {
               this.sendSelect(item.getId(),callback);
               return;
            }
         }
         callback();
      }
      
      private function saveArtifact(callback:Function) : void
      {
         var item:HatRecord = Boolean(this.chosenArtifact) ? this.chosenArtifact.getRecord() as HatRecord : null;
         if(this.p == Application.userProfile && item != this.p.getArtifact())
         {
            // HatRecord.ARTIFACT_START_ID (no item) takes the artifact off.
            this.sendSelect(Boolean(item) ? item.getId() : HatRecord.ARTIFACT_START_ID,callback);
            return;
         }
         callback();
      }
      
      private function sendSelect(stuffId:int, callback:Function) : void
      {
         SocketEvents.single(MainConnection.INSTANCE,SelectStuffResult,this.onSelectResult,this,[callback],true);
         Message.SAVING_MSG.show();
         MainConnection.sendMessage(new SelectStuff(stuffId));
      }
      
      private function onSelectResult(e:IncommingMessageEvent, callback:Function) : void
      {
         var msgList:Object = null;
         var stuffId:int = 0;
         Message.SAVING_MSG.hide();
         if(SelectStuffResult(e.message).result == SelectStuffResult.SUCCESS)
         {
            stuffId = SelectStuffResult(e.message).stuffId;
            if(stuffId >= HatRecord.ARTIFACT_START_ID)
            {
               this.p.setArtifact(ItemsStore.getArtifactById(stuffId));
            }
            else
            {
               this.p.setHat(ItemsStore.getHatById(stuffId,Application.userProfile.getCharType()));
            }
            Application.mainMenu.updateUserInfo();
         }
         else
         {
            msgList = {};
            msgList[SelectStuffResult.ERROR] = Messages.ERROR_SERVER;
            MessageUtils.showErrorMsgWithSound(UIMessages.TITLE_ATTENTION,SelectStuffResult(e.message).result,msgList);
         }
         callback();
      }
      
      override protected function createCell(cell:DisplayObject) : IUIMenuItem
      {
         return new WardrobeItem(cell);
      }
      
      public function initFromProfile(p:UserProfile) : void
      {
         var item:BackpackItem = null;
         this.p = p;
         this.chosenHat = null;
         this.chosenArtifact = null;
         // Hats first, then artifacts.
         var hats:Array = [];
         var artifacts:Array = [];
         for each(item in p.getWardrobe().getDepositSortedArray())
         {
            if(HatRecord(item.getRecord()).isArtifact)
            {
               artifacts.push(item);
               if(item.getRecord() == p.getArtifact())
               {
                  this.chosenArtifact = item;
               }
            }
            else
            {
               hats.push(item);
               if(item.getRecord() == p.getHat())
               {
                  this.chosenHat = item;
               }
            }
         }
         this.data = hats.concat(artifacts);
         setData(this.data);
         if(Boolean(this.chosenHat))
         {
            setSelectedData(this.chosenHat);
         }
         else if(Boolean(this.chosenArtifact))
         {
            setSelectedData(this.chosenArtifact);
         }
         this.charIcon = UserHint.getCharIcon(this.charIcon,p,getChildByNameAsMovieClip("charIcon"));
         this.updateSlots();
      }
      
      override protected function initCell(cell:IUIMenuItem, item:Object, dataIndex:uint) : void
      {
         cell.setItem(item);
         cell.setVisible(item != null);
         if(Boolean(item))
         {
            cell.setSelectable(this.p == Application.userProfile && Boolean(item));
         }
         // The menu highlights one cell: mark the other slot's item too (pages are redrawn).
         cell.setSelected(Boolean(item) && (item == this.chosenHat || item == this.chosenArtifact));
      }
      
      override protected function onCellClick(e:Event) : void
      {
         var item:BackpackItem = IUIMenuItem(e.target).getItem() as BackpackItem;
         if(Boolean(item) && this.p == Application.userProfile)
         {
            if(HatRecord(item.getRecord()).isArtifact)
            {
               this.chosenArtifact = item == this.chosenArtifact ? null : item;
            }
            else
            {
               this.chosenHat = item;
            }
         }
         super.onCellClick(e);
         // The cell selects itself after this click event: fix the highlights once it's done.
         setTimeout(this.updateSlots,1);
      }
      
      override protected function updateItemInfo(value:Object) : void
      {
         this.updateSlots();
      }
      
      private function updateSlots() : void
      {
         var item:BackpackItem = null;
         var cell:IUIMenuItem = null;
         for each(item in this.data)
         {
            cell = getCellByItem(item);
            if(Boolean(cell))
            {
               cell.setSelected(item == this.chosenHat || item == this.chosenArtifact);
            }
         }
         this.setHat(Boolean(this.chosenHat) ? this.chosenHat.getRecord() as HatRecord : null);
      }
      
      private function setHat(hat:HatRecord) : void
      {
         if(Boolean(hat) && hat.canBeUsed(this.p.getCharType()))
         {
            this.charIcon.setHat(hat);
         }
         else
         {
            this.charIcon.setHat(null);
         }
         this.charIcon.breath(false);
      }
      
      public function getSwfName() : String
      {
         return this.nameSwf;
      }
      
      public function getContainerClass() : String
      {
         return Classes.WARDROBE;
      }
   }
}

